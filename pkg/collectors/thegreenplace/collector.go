// Package thegreenplace implement collector for https://eli.thegreenplace.net/
package thegreenplace

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
		"vela.collectors.thegreenplace",
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
	return "thegreenplace"
}

// Initialize implement collector.Initialize
func (c *Collector) Initialize(_ context.Context) error {
	return nil
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- collectors.Post) error {
	defer close(ch)

	c.listCollector.OnHTML("table.archive-list", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("tbody > tr", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("td a", "href")
			if path == "" {
				return false
			}

			slogctx.FromCtx(ctx).InfoContext(ctx, "collect article", slog.String("Path", path))
			err := c.postCollector.Request("GET", path, nil, nil, nil)
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

	c.postCollector.OnHTML("#content article", func(h *colly.HTMLElement) {
		title := h.ChildText("header.page-header h1 a")
		if title == "" {
			return
		}

		publishedAt := h.ChildText("span.published time")
		t, err := time.Parse("January 2, 2006 at 15:04", publishedAt)
		if err != nil {
			return
		}

		postBody, err := h.DOM.Find("div.entry-content").Html()
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
			PublishedAt: t,
			Content:     bodyMarkdown,
		}
	})

	err := c.listCollector.Request("GET", "https://eli.thegreenplace.net/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	c.postCollector.Wait()
	return nil
}
