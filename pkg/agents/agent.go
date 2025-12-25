// Package agents contains implementation of llm summarizer.
package agents

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/anyvoxel/airmid/anvil"
	"github.com/anyvoxel/airmid/anvil/xerrors"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"
	"github.com/anyvoxel/vela/pkg/apitypes"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

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

	a.agent = agent
	a.systemPrompt = systemPrompts
	return nil
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
	err = chromedp.Run(ctx,
		chromedp.Navigate(post.Path),
		chromedp.Sleep(10*time.Second),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err = page.PrintToPDF().Do(ctx)
			return err
		}),
	)
	_ = buf

	if err != nil {
		return "", err
	}

	// TODO: upload buf to aliyun oss
	prompt := blades.UserMessage(blades.TextPart{
		Text: fmt.Sprintf("Please summarize the following blog post in the pdf {%s}", post.Path),
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
	err = chromedp.Run(ctx,
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
	_ = buf

	if err != nil {
		return "", err
	}

	// TODO: upload buf to aliyun oss
	prompt := blades.UserMessage(blades.FilePart{
		Name:     "image_url",
		URI:      post.Path,
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
