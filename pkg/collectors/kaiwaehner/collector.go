// Package kaiwaehner implement collector for https://www.kai-waehner.de/
package kaiwaehner

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
		"vela.collectors.kaiwaehner",
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
	return "kaiwaehner"
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

	postCollector.OnHTML("#content", func(h *colly.HTMLElement) {
		var postBody string
		postBody, err = h.DOM.Find("div.entry-content div[dir=auto]").Html()
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

	c.listCollector.OnHTML("div.post-archive div.archive-main", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("article.post", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("h2.entry-title a", "href")
			if path == "" {
				return true
			}

			title := h.ChildText("h2.entry-title a")
			publishedAt := h.ChildText("ul.post-meta li.meta-date")
			t, err := time.Parse("2. January 2006", publishedAt)
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

	err := c.listCollector.Request("GET", "https://www.kai-waehner.de/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
