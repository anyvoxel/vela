// Package sebastianraschka implement collector for https://sebastianraschka.com/blog/
package sebastianraschka

import (
	"context"
	"log/slog"
	"reflect"
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
		"vela.collectors.sebastianraschka",
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
	return "sebastianraschka"
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

	postCollector.OnHTML("div.page-content", func(h *colly.HTMLElement) {
		var postBody string
		postBody, err = h.DOM.Find("article.post-content").Html()
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

	c.listCollector.OnHTML("article.post-content div.posts-by-year", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("li", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("a", "href")
			if path == "" {
				return true
			}
			path = "https://sebastianraschka.com" + path

			title := h.ChildAttr("a", "title")
			publishedAt := h.ChildText("span.post-date")
			t, err := time.Parse("Jan 2, 2006", publishedAt)
			if err != nil {
				slogctx.FromCtx(ctx).ErrorContext(ctx,
					"parse datetime failed",
					slog.String("Path", path),
					slog.Any("Error", err),
				)
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

	err := c.listCollector.Request("GET", "https://sebastianraschka.com/blog/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
