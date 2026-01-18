// Package cameronrwolfe implement collector for https://cameronrwolfe.substack.com/archive?sort=new
package cameronrwolfe

import (
	"context"
	"log/slog"
	"reflect"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/anyvoxel/airmid/anvil"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"
	"github.com/gocolly/colly/v2"
	slogctx "github.com/veqryn/slog-context"

	"github.com/anyvoxel/vela/pkg/apitypes"
	"github.com/anyvoxel/vela/pkg/collectors"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.collectors.cameronrwolfe",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Collector)(nil)),
		),
	))
}

// Collector implemetation.
type Collector struct {
	listCollector *colly.Collector
}

var (
	_ ioc.InitializingBean = (*Collector)(nil)
	_ collectors.Collector = (*Collector)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (c *Collector) AfterPropertiesSet(context.Context) error {
	c.listCollector = colly.NewCollector()
	return nil
}

// Name implement collector.Name
func (c *Collector) Name() string {
	return "cameronrwolfes"
}

// Initialize implement collector.Initialize
func (c *Collector) Initialize(_ context.Context) error {
	return nil
}

// ResolvePostContent implement collector.ResolvePostContent
func (c *Collector) ResolvePostContent(_ context.Context, post apitypes.Post) (string, error) {
	postCollector := colly.NewCollector()
	var bodyMarkdown string
	var err error

	postCollector.OnHTML("body", func(h *colly.HTMLElement) {
		var postBody string
		postBody, err = h.DOM.Find("#main").Html()
		if err != nil {
			return
		}
		bodyMarkdown, err = md.NewConverter("", true, nil).ConvertString(postBody)
		if err != nil {
			return
		}
	})

	err = postCollector.Request("GET", post.Path, nil, nil, nil)
	if err != nil {
		return "", err
	}

	postCollector.Wait()
	return bodyMarkdown, err
}

func (c *Collector) extracePublishedAt(ctx context.Context, h *colly.HTMLElement) time.Time {
	publishedAt := h.ChildAttr("div div.pencraft time", "datetime")
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(publishedAt))
	if err != nil {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"parse datetime failed",
			slog.Any("Error", err),
			slog.String("PublishedAt", publishedAt),
		)
		return time.Time{}
	}
	return t
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- apitypes.Post) error {
	c.listCollector.OnHTML("div.portable-archive div.portable-archive-list", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("div[role=article]", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("a[data-testid=post-preview-title]", "href")
			if path == "" {
				return true
			}

			title := h.ChildText("a[data-testid=post-preview-title]")
			t := c.extracePublishedAt(slogctx.With(ctx, slog.String("Path", path)), h)

			slogctx.FromCtx(ctx).InfoContext(ctx, "collect article",
				slog.String("Path", path),
				slog.String("Title", title),
				slog.Any("PublishedAt", t),
			)
			post := apitypes.Post{
				Domain:      c.Name(),
				Title:       title,
				Path:        path,
				PublishedAt: t,
			}
			ch <- post

			return true
		})
	})

	err := c.listCollector.Request("GET",
		"https://cameronrwolfe.substack.com/archive?sort=new", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
