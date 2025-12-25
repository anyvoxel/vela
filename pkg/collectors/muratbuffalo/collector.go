// Package muratbuffalo implement collector for muratbuffalo.blogspot.com
package muratbuffalo

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
		"vela.collectors.muratbuffalo",
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
	return "muratbuffalo"
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

	postCollector.OnHTML("div.post", func(h *colly.HTMLElement) {
		var postBody string
		postBody, err = h.DOM.Find("div.post-body").Html()
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

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- apitypes.Post) error {
	defer close(ch)

	c.listCollector.OnHTML("div.Blog", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("div.blog-posts div.post", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("h3.post-title a", "href")
			if path == "" {
				return true
			}

			title := h.ChildText("h3.post-title a")
			publishedAt := h.ChildText("div.post-header span.post-timestamp time.published")
			t, err := time.Parse("January 02, 2006", strings.TrimSpace(publishedAt))
			if err != nil {
				slogctx.FromCtx(ctx).ErrorContext(ctx,
					"parse datetime failed",
					slog.Any("Error", err),
					slog.String("Path", path))
				return true
			}

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

	err := c.listCollector.Request("GET", "https://muratbuffalo.blogspot.com", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
