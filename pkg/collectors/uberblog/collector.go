// Package uberblog implement collector for https://www.uber.com/en-SG/blog/
package uberblog

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/gocolly/colly/v2"

	"github.com/anyvoxel/vela/pkg/collectors"
)

// Collector implemetation.
type Collector struct {
	listCollector *colly.Collector
	postCollector *colly.Collector
	header        http.Header
}

// NewCollector creates an new implementation.
func NewCollector(_ context.Context) (collectors.Collector, error) {
	header := http.Header{}
	header.Add("Content-Type", "text/html; charset=utf-8")
	header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7") //nolint
	header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")               //nolint

	return &Collector{
		listCollector: colly.NewCollector(),
		postCollector: colly.NewCollector(),
		header:        header,
	}, nil
}

// Name implement collector.Name
func (c *Collector) Name() string {
	return "uberblog"
}

// Initialize implement collector.Initialize
func (c *Collector) Initialize(_ context.Context) error {
	return nil
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- collectors.Post) error {
	defer close(ch)

	c.listCollector.OnHTML("#main", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("div.cs", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("a.dd", "href")
			if path == "" {
				return false
			}

			slog.InfoContext(ctx, "collect article", slog.String("Path", path))
			err := c.postCollector.Request("GET", "https://www.uber.com"+path, nil, nil, c.header)
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

	c.postCollector.OnHTML("#main", func(h *colly.HTMLElement) {
		title := h.ChildText("header div h1.ev")
		if title == "" {
			return
		}

		postBody, err := h.DOM.Find("div.ga").Html()
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

	err := c.listCollector.Request("GET", "https://www.uber.com/en-SG/blog/", nil, colly.NewContext(), c.header)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	c.postCollector.Wait()
	return nil
}
