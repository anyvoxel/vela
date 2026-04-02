package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anyvoxel/airmid/anvil"
	"github.com/anyvoxel/airmid/anvil/xerrors"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"
	"github.com/anyvoxel/vela/pkg/apitypes"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.agents.listParser",
		ioc.MustNewBeanDefinition(
			reflect.TypeFor[*listParserImpl](),
		),
	))
}

const listParserSystemPrompt = `You are a precise HTML list parser.
Return ONLY valid JSON object with this schema:
{"error":"", "items":[{"url":"", "title":"", "published_at":""}]}

Rules:
- Extract only blog/article list items from the HTML.
- Resolve relative URLs to absolute using the provided BaseURL.
- The url must be a canonical absolute URL.
- title should be the article title if available, otherwise empty string.
- published_at should be RFC3339 (e.g. 2026-03-30T08:30:00Z) or YYYY-MM-DD if time is unknown.
- If no items are found, return {"error":"", "items":[]}.
- Never include any explanation or extra text.
`

// listParserImpl is an agent that extracts list items from HTML.
type listParserImpl struct {
	chatModel    *openai.ChatModel
	systemPrompt string
}

var (
	_ ioc.InitializingBean = (*listParserImpl)(nil)
)

var errListParserResponse = errors.New("list parser response error")

var errBaseURLMustBeAbsolute = errors.New("base url must be absolute")

// AfterPropertiesSet implement InitializingBean
func (a *listParserImpl) AfterPropertiesSet(ctx context.Context) error {
	responseFormat := &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:         os.Getenv("OPENAI_API_KEY_LIST_PARSER"),
		Model:          os.Getenv("OPENAI_MODEL_LIST_PARSER"),
		BaseURL:        os.Getenv("OPENAI_BASE_URL_LIST_PARSER"),
		ResponseFormat: responseFormat,
		ByAzure:        os.Getenv("OPENAI_BY_AZURE_LIST_PARSER") == "true",
	})
	if err != nil {
		return err
	}

	a.chatModel = chatModel
	a.systemPrompt = listParserSystemPrompt
	return nil
}

type listParseItem struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	PublishedAt string `json:"published_at"`
}

type listParseResult struct {
	Error string          `json:"error"`
	Items []listParseItem `json:"items"`
}

// ParseList extracts post metadata from list page HTML.
func (a *listParserImpl) ParseList(ctx context.Context, html, baseURL, domain string) ([]apitypes.Post, error) {
	resolveBaseURL, err := normalizeResolveBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	message := &schema.Message{
		Role: schema.User,
		Content: fmt.Sprintf("BaseURL: %s\nDomain: %s\nHTML:\n%s",
			resolveBaseURL, domain, html),
	}

	text, err := a.generate(ctx, message)
	if err != nil {
		return nil, err
	}

	result, err := parseListResult(text)
	if err != nil {
		return nil, err
	}

	return buildPosts(result, resolveBaseURL, domain)
}

func parseListResult(text string) (listParseResult, error) {
	var result listParseResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		slog.Default().Error(
			"unmarshal list parser result failed",
			slog.Any("Error", err),
			slog.Int("TextLen", len(text)),
			slog.String("Text", truncateForLog(text, 8*1024)),
		)
		return result, err
	}
	if result.Error != "" {
		return result, fmt.Errorf("%w: %s", errListParserResponse, result.Error)
	}
	return result, nil
}

func truncateForLog(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return s
	}
	if len(s) <= maxBytes {
		return s
	}

	b := []byte(s)
	cut := maxBytes
	if cut > len(b) {
		cut = len(b)
	}
	for cut > 0 && !utf8.Valid(b[:cut]) {
		cut--
	}
	return string(b[:cut]) + "...(truncated)"
}

func buildPosts(result listParseResult, baseURL, domain string) ([]apitypes.Post, error) {
	baseURL, err := normalizeResolveBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(result.Items))
	posts := make([]apitypes.Post, 0, len(result.Items))
	for _, item := range result.Items {
		post, ok := buildPost(item, baseParsed, domain, seen)
		if !ok {
			continue
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// normalizeResolveBaseURL converts an absolute URL (which may include a path) into an origin URL
// (scheme + host, trailing slash) so that resolving relative post links won't accidentally inherit
// the list page path.
func normalizeResolveBaseURL(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	if !parsed.IsAbs() {
		return "", fmt.Errorf("%w: %q", errBaseURLMustBeAbsolute, baseURL)
	}

	origin := &url.URL{
		Scheme: parsed.Scheme,
		User:   parsed.User,
		Host:   parsed.Host,
		Path:   "/",
	}
	return origin.String(), nil
}

func buildPost(item listParseItem, baseParsed *url.URL, domain string, seen map[string]struct{}) (apitypes.Post, bool) {
	itemURL := strings.TrimSpace(item.URL)
	if itemURL == "" {
		return apitypes.Post{}, false
	}

	parsed, err := url.Parse(itemURL)
	if err != nil {
		return apitypes.Post{}, false
	}
	if !parsed.IsAbs() {
		parsed = baseParsed.ResolveReference(parsed)
	}
	itemURL = parsed.String()
	if _, ok := seen[itemURL]; ok {
		return apitypes.Post{}, false
	}
	seen[itemURL] = struct{}{}

	return apitypes.Post{
		Domain:      domain,
		Title:       strings.TrimSpace(item.Title),
		Path:        itemURL,
		PublishedAt: parsePublishedAt(item.PublishedAt),
	}, true
}

func parsePublishedAt(publishedAt string) time.Time {
	if strings.TrimSpace(publishedAt) == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, publishedAt); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", publishedAt); err == nil {
		return t
	}
	return time.Time{}
}

func (a *listParserImpl) generate(ctx context.Context, userMessage *schema.Message) (string, error) {
	resp, err := a.chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: a.systemPrompt,
		},
		userMessage,
	})
	if err != nil {
		return "", err
	}
	text := extractMessageText(resp)
	if text == "" {
		return "", xerrors.Errorf("empty response from llm")
	}
	return text, nil
}
