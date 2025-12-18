// Package simonwillison implement collector for https://simonwillison.net/
package simonwillison

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

	"github.com/anyvoxel/vela/pkg/collectors"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.collectors.simonwillison",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Collector)(nil)),
		),
	))
}

// Collector implemetation.
type Collector struct {
	listCollector *colly.Collector
	postCollector *colly.Collector
}

var (
	_ ioc.InitializingBean = (*Collector)(nil)
	_ collectors.Collector = (*Collector)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (c *Collector) AfterPropertiesSet(context.Context) error {
	c.listCollector = colly.NewCollector()
	c.postCollector = colly.NewCollector()
	return nil
}

// Name implement collector.Name
func (c *Collector) Name() string {
	return "simonwillison"
}

// Initialize implement collector.Initialize
func (c *Collector) Initialize(_ context.Context) error {
	return nil
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- collectors.Post) error {
	defer close(ch)

	c.listCollector.OnHTML("#primary", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("div.entry", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("h3 a", "href")
			if path == "" {
				return false
			}

			slogctx.FromCtx(ctx).InfoContext(ctx, "collect article", slog.String("Path", path))
			err := c.postCollector.Request("GET", "https://simonwillison.net"+path, nil, nil, nil)
			if err != nil {
				slogctx.FromCtx(ctx).ErrorContext(ctx,
					"cann't request on article",
					slog.Any("Error", err),
					slog.String("NextPath", path))
				return false
			}

			return true
		})
	})

	c.postCollector.OnHTML("#primary div.entry", func(h *colly.HTMLElement) {
		title := h.DOM.Children().First().ChildrenFiltered("h2").Text()
		if title == "" {
			return
		}

		postBody, err := h.DOM.Children().First().Html()
		if err != nil {
			return
		}
		bodyMarkdown, err := md.NewConverter("", true, nil).ConvertString(postBody)
		if err != nil {
			return
		}
		ch <- collectors.Post{
			Domain:      c.Name(),
			Title:       title,
			Path:        h.Request.URL.String(),
			PublishedAt: time.Time{},
			Content:     bodyMarkdown,
		}
	})

	err := c.listCollector.Request("GET", "https://simonwillison.net/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	c.postCollector.Wait()
	return nil
}
