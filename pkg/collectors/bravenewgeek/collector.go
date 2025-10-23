// Package bravenewgeek implement collector for https://bravenewgeek.com/
package bravenewgeek

import (
	"context"
	"log/slog"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/gocolly/colly/v2"

	"github.com/anyvoxel/vela/pkg/collectors"
)

// Collector implemetation.
type Collector struct {
	listCollector *colly.Collector
	postCollector *colly.Collector
}

// NewCollector creates an new implementation.
func NewCollector(_ context.Context) (collectors.Collector, error) {
	return &Collector{
		listCollector: colly.NewCollector(),
		postCollector: colly.NewCollector(),
	}, nil
}

// Name implement collector.Name
func (c *Collector) Name() string {
	return "bravenewgeek"
}

// Initialize implement collector.Initialize
func (c *Collector) Initialize(_ context.Context) error {
	return nil
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- collectors.Post) error {
	defer close(ch)

	c.listCollector.OnHTML("#main ul.display-posts-listing", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("li.listing-item", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("a.title", "href")
			if path == "" {
				return false
			}

			slog.InfoContext(ctx, "collect article", slog.String("Path", path))
			err := c.postCollector.Request("GET", path, nil, nil, nil)
			if err != nil {
				slog.ErrorContext(ctx,
					"cann't request on article",
					slog.Any("Error", err),
					slog.String("NextPath", path))
				return false
			}

			return true
		})
	})

	c.postCollector.OnHTML("#main article", func(h *colly.HTMLElement) {
		title := h.ChildText("header h1.entry-title")
		if title == "" {
			return
		}

		publishedAt := h.ChildText("header span.posted-on a time.published")
		t, err := time.Parse("Jan 2, 2006", publishedAt)
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

	err := c.listCollector.Request("GET", "https://bravenewgeek.com/archive/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	c.postCollector.Wait()
	return nil
}
