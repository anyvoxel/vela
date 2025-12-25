// Package agents contains implementation of llm summarizer.
package agents

import (
	"bytes"
	"context"
	_ "embed"
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

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/contrib/openai"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.agents.summarizer",
		ioc.MustNewBeanDefinition(
			reflect.TypeFor[*Summarizer](),
		),
	))
}

//go:embed system_prompts.md
var systemPrompts string

// Summarizer is a agent that can summarize a blog post.
type Summarizer struct {
	agent         blades.Agent
	summarizeType string `airmid:"value:${vela.summarize.type:=image}"`
	systemPrompt  string

	ossRegion string `airmid:"value:${vela.summarize.oss.region:=cn-beijing}"`
	ossBucket string `airmid:"value:${vela.summarize.oss.bucket:=anyvoxel-vela}"`
	ossClient *oss.Client

	// Summary summarizes the given content.
	Summary func(ctx context.Context, post apitypes.Post) (string, error)
}

var (
	_ ioc.InitializingBean = (*Summarizer)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (a *Summarizer) AfterPropertiesSet(context.Context) error {
	model := openai.NewModel(os.Getenv("OPENAI_MODEL"), openai.Config{
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	agent, err := blades.NewAgent("Summary Agent",
		blades.WithModel(model), blades.WithInstruction(systemPrompts))
	if err != nil {
		return err
	}

	switch a.summarizeType {
	case "image":
		a.Summary = a.summarizeByImage
	case "pdf":
		a.Summary = a.summarizeByPdf
	case "text":
		a.Summary = a.summarizeByText
	default:
		return xerrors.Errorf("Unknown summary type: %s", a.summarizeType)
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(a.ossRegion)
	a.ossClient = oss.NewClient(cfg)

	a.agent = agent
	a.systemPrompt = systemPrompts
	return nil
}

func (a *Summarizer) putFileToOSS(ctx context.Context, filename string, data []byte) (string, func(), error) {
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

func (a *Summarizer) summarizeByText(ctx context.Context, post apitypes.Post) (string, error) {
	if post.ContentResolver == nil {
		return "", xerrors.Errorf("cann't got content with nil resolver, domain: %s, path: %s", post.Domain, post.Path)
	}

	content, err := post.ContentResolver()
	if err != nil {
		return "", fmt.Errorf("got content failed: %w, domain: %s, path: %s", err, post.Domain, post.Path)
	}
	if content == "" {
		return "", xerrors.Errorf("got empty content, domain: %s, path: %s", post.Domain, post.Path)
	}

	prompt := blades.UserMessage(
		fmt.Sprintf("Please summarize the following blog post (with Markdown or HTML format): %s",
			content),
	)
	runner := blades.NewRunner(a.agent)
	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}

func (a *Summarizer) summarizeByPdf(ctx context.Context, post apitypes.Post) (string, error) {
	var buf []byte
	var err error

	dpctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	err = chromedp.Run(dpctx,
		chromedp.Navigate(post.Path),
		chromedp.Sleep(10*time.Second),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err = page.PrintToPDF().Do(ctx)
			return err
		}),
	)

	if err != nil {
		return "", err
	}

	path, clean, err := a.putFileToOSS(ctx, post.Path, buf)
	if err != nil {
		return "", err
	}
	defer clean()

	prompt := blades.UserMessage(blades.TextPart{
		Text: fmt.Sprintf("Please summarize the following blog post in the pdf {%s}", path),
	})

	runner := blades.NewRunner(a.agent)
	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}

func (a *Summarizer) summarizeByImage(ctx context.Context, post apitypes.Post) (string, error) {
	var buf []byte
	var err error

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("headless", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	dpctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	err = chromedp.Run(dpctx,
		chromedp.Navigate(post.Path),
		chromedp.Sleep(10*time.Second),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, err = page.CaptureScreenshot().
				WithQuality(90).
				WithCaptureBeyondViewport(true).
				WithFromSurface(true).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		return "", err
	}

	path, clean, err := a.putFileToOSS(ctx, post.Path, buf)
	if err != nil {
		return "", err
	}
	defer clean()

	prompt := blades.UserMessage(blades.FilePart{
		Name:     "image_url",
		URI:      path,
		MIMEType: blades.MIMEImagePNG,
	}, blades.TextPart{
		Text: "Please summarize the following blog post in the image",
	})

	runner := blades.NewRunner(a.agent)
	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}
