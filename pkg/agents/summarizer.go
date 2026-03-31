package agents

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/anyvoxel/airmid/anvil"
	"github.com/anyvoxel/airmid/anvil/xerrors"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"
	"github.com/anyvoxel/vela/pkg/apitypes"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	slogctx "github.com/veqryn/slog-context"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.agents.summarizer",
		ioc.MustNewBeanDefinition(
			reflect.TypeFor[*summarizerImpl](),
		),
	))
}

//go:embed system_prompts.md
var systemPrompts string

// Summarizer is the interface for summarizer.
type Summarizer interface {
	// Summary summarizes the given content.
	Summary(ctx context.Context, post apitypes.Post) (string, error)
}

// summarizerImpl is a agent that can summarize a blog post.
type summarizerImpl struct {
	chatModel     *openai.ChatModel
	summarizeType string `airmid:"value:${vela.summarize.type:=image}"`
	systemPrompt  string

	ossRegion string `airmid:"value:${vela.summarize.oss.region:=cn-beijing}"`
	ossBucket string `airmid:"value:${vela.summarize.oss.bucket:=anyvoxel-vela}"`
	ossClient *oss.Client

	summaryFn func(ctx context.Context, post apitypes.Post) (string, error)
}

var (
	_ ioc.InitializingBean = (*summarizerImpl)(nil)
	_ Summarizer           = (*summarizerImpl)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (a *summarizerImpl) AfterPropertiesSet(ctx context.Context) error {
	responseFormat := &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:         os.Getenv("OPENAI_API_KEY"),
		Model:          os.Getenv("OPENAI_MODEL"),
		BaseURL:        os.Getenv("OPENAI_BASE_URL"),
		ResponseFormat: responseFormat,
		ByAzure:        os.Getenv("OPENAI_BY_AZURE") == "true",
	})
	if err != nil {
		return err
	}

	switch a.summarizeType {
	case "image":
		a.summaryFn = a.summarizeByImage
	case "pdf":
		a.summaryFn = a.summarizeByPdf
	default:
		return xerrors.Errorf("Unknown summary type: %s", a.summarizeType)
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(a.ossRegion)
	a.ossClient = oss.NewClient(cfg)

	a.chatModel = chatModel
	a.systemPrompt = systemPrompts
	return nil
}

type generateResult struct {
	Error   string `json:"error"`
	Summary string `json:"summary"`
}

// Summary implement Summarizer.Summary
func (a *summarizerImpl) Summary(ctx context.Context, post apitypes.Post) (string, error) {
	text, err := a.summaryFn(ctx, post)
	if err != nil {
		return "", err
	}

	slogctx.FromCtx(ctx).InfoContext(ctx,
		"generate llm output", slog.String("Output", text))
	var result generateResult
	err = json.Unmarshal([]byte(text), &result)
	if err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", errors.New(result.Error) //nolint
	}

	return result.Summary, nil
}

func (a *summarizerImpl) putFileToOSS(ctx context.Context, filename string, data []byte) (string, func(), error) {
	// Result looks like:
	// {
	// 	"ContentMD5": "CY9rzUYh03PK3k6DJie09g==",
	// 	"ETag": "\"098F6BCD4621D373CADE4E832627B4F6\"",
	// 	"HashCRC64": "18020588380933092773",
	// 	"VersionId": null,
	// 	"CallbackResult": null,
	// 	"Status": "200 OK",
	// 	"StatusCode": 200,
	// 	"Headers": {
	// 		"Connection": ["keep-alive"],
	// 		"Content-Length": ["0"],
	// 		"Content-Md5": ["CY9rzUYh03PK3k6DJie09g=="],
	// 		"Date": ["Thu, 25 Dec 2025 07:11:47 GMT"],
	// 		"Etag": ["\"098F6BCD4621D373CADE4E832627B4F6\""],
	// 		"Server": ["AliyunOSS"],
	// 		"X-Oss-Hash-Crc64ecma": ["18020588380933092773"],
	// 		"X-Oss-Request-Id": ["694CE3B3C08C163936CEF268"],
	// 		"X-Oss-Server-Time": ["64"]
	// 	},
	// 	"OpMetadata": {}
	// }
	result, err := a.ossClient.PutObject(ctx,
		&oss.PutObjectRequest{
			Bucket: oss.Ptr(a.ossBucket),
			Key:    oss.Ptr(filename),
			Body:   bytes.NewReader(data),
		})
	if err != nil {
		return "", nil, err
	}

	slogctx.FromCtx(ctx).InfoContext(ctx,
		"put file to oss success",
		slog.String("Filename", filename),
		slog.Any("ContentMD5", result.ContentMD5),
		slog.Any("Version", result.VersionId),
	)

	return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", a.ossBucket, a.ossRegion, filename), func() {
		result, err := a.ossClient.DeleteObject(ctx, &oss.DeleteObjectRequest{
			Bucket: oss.Ptr(a.ossBucket),
			Key:    oss.Ptr(filename),
		})
		if err != nil {
			slogctx.FromCtx(ctx).ErrorContext(ctx,
				"delete file from oss failed",
				slog.String("Filename", filename),
			)
			return
		}

		slogctx.FromCtx(ctx).InfoContext(ctx,
			"delete file from oss success",
			slog.String("Filename", filename),
			slog.Any("Version", result.VersionId),
		)
	}, nil
}

func (a *summarizerImpl) runActionInChrome(ctx context.Context, path string, fn chromedp.ActionFunc) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("headless", true),
		//nolint
		// See https://stackoverflow.com/questions/70535305/getting-403-forbidden-error-when-using-headless-chrome-with-python-selenium
		chromedp.Flag("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.50 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	dpctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	return chromedp.Run(dpctx,
		chromedp.Navigate(path),
		chromedp.Sleep(10*time.Second),
		chromedp.WaitReady("body"),
		fn,
	)
}

func (a *summarizerImpl) summarizeByPdf(ctx context.Context, post apitypes.Post) (string, error) {
	var buf []byte
	var err error

	err = a.runActionInChrome(ctx, post.Path, chromedp.ActionFunc(func(ctx context.Context) error {
		buf, _, err = page.PrintToPDF().Do(ctx)
		return err
	}))

	if err != nil {
		return "", err
	}

	path, clean, err := a.putFileToOSS(ctx, post.Path, buf)
	if err != nil {
		return "", err
	}
	defer clean()

	message := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("Please summarize the following blog post in the pdf {%s}", path),
	}
	return a.generate(ctx, message)
}

func (a *summarizerImpl) summarizeByImage(ctx context.Context, post apitypes.Post) (string, error) {
	var buf []byte
	var err error

	err = a.runActionInChrome(ctx, post.Path, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, err = page.CaptureScreenshot().
			WithQuality(90).
			WithCaptureBeyondViewport(true).
			WithFromSurface(true).
			Do(ctx)
		return err
	}))

	if err != nil {
		return "", err
	}

	path, clean, err := a.putFileToOSS(ctx, post.Path, buf)
	if err != nil {
		return "", err
	}
	defer clean()

	message := &schema.Message{
		Role: schema.User,
		UserInputMultiContent: []schema.MessageInputPart{
			{
				Type: schema.ChatMessagePartTypeText,
				Text: "Please summarize the following blog post in the image",
			},
			{
				Type: schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						URL: &path,
					},
					Detail: schema.ImageURLDetailAuto,
				},
			},
		},
	}
	return a.generate(ctx, message)
}

func (a *summarizerImpl) generate(ctx context.Context, userMessage *schema.Message) (string, error) {
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
