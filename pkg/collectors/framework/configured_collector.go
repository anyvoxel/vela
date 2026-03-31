package framework

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gocolly/colly/v2"
	slogctx "github.com/veqryn/slog-context"

	"github.com/anyvoxel/vela/pkg/apitypes"
	"github.com/anyvoxel/vela/pkg/collectors"
)

var (
	errCollectorNameEmpty  = errors.New("collector name is empty")
	errCollectorURLEmpty   = errors.New("collector url is empty")
	errListParserNil       = errors.New("listParser is nil")
	errURLMustBeAbsolute   = errors.New("url must be absolute")
	errCollectorURLInvalid = errors.New("collector url is invalid")
)

// CollectorSource describes a list page source that can be collected.
// It is intended to be loaded from JSON config.
type CollectorSource struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type configuredCollector struct {
	name       string
	url        string
	header     http.Header
	listParser collectors.ListParser
}

func newConfiguredCollector(src CollectorSource, listParser collectors.ListParser) (*configuredCollector, error) {
	name := strings.TrimSpace(src.Name)
	if name == "" {
		return nil, errCollectorNameEmpty
	}
	urlStr := strings.TrimSpace(src.URL)
	if urlStr == "" {
		return nil, errCollectorURLEmpty
	}
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", errCollectorURLInvalid, urlStr, err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("%w: %q", errURLMustBeAbsolute, urlStr)
	}
	if listParser == nil {
		return nil, errListParserNil
	}

	var hdr http.Header
	if len(src.Headers) > 0 {
		hdr = make(http.Header, len(src.Headers))
		for k, v := range src.Headers {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			hdr.Set(k, strings.TrimSpace(v))
		}
	}

	return &configuredCollector{
		name:       name,
		url:        parsed.String(),
		header:     hdr,
		listParser: listParser,
	}, nil
}

func (c *configuredCollector) Name() string { return c.name }

func (c *configuredCollector) Initialize(_ context.Context) error { return nil }

func (c *configuredCollector) Start(ctx context.Context, ch chan<- apitypes.Post) error {
	listCollector := colly.NewCollector()
	listCollector.OnResponse(func(r *colly.Response) {
		posts, err := c.listParser.ParseList(ctx, string(r.Body), r.Request.URL.String(), c.Name())
		if err != nil {
			slogctx.FromCtx(ctx).ErrorContext(ctx,
				"parse list failed",
				slog.Any("Error", err),
				slog.String("URL", r.Request.URL.String()),
			)
			return
		}
		for _, post := range posts {
			slogctx.FromCtx(ctx).InfoContext(ctx, "collect article",
				slog.String("Path", post.Path),
				slog.String("Title", post.Title),
				slog.Any("PublishedAt", post.PublishedAt),
			)
			ch <- post
		}
	})

	err := listCollector.Request("GET", c.url, nil, colly.NewContext(), c.header)
	if err != nil {
		return err
	}
	listCollector.Wait()
	return nil
}
