// Package micahlerner implement collector for https://www.micahlerner.com/
package micahlerner

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

	"github.com/anyvoxel/vela/pkg/collectors"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.collectors.micahlerner",
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
	return "micahlerner"
}

// Initialize implement collector.Initialize
func (c *Collector) Initialize(_ context.Context) error {
	return nil
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- collectors.Post) error {
	defer close(ch)

	c.listCollector.OnHTML("body article section div ul", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("li", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("p a", "href")
			if path == "" {
				return false
			}

			slog.InfoContext(ctx, "collect article", slog.String("Path", path))
			err := c.postCollector.Request("GET", "https://www.micahlerner.com"+path, nil, nil, nil)
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

	c.postCollector.OnHTML("body article", func(h *colly.HTMLElement) {
		title := h.ChildText("h1")
		if title == "" {
			return
		}

		publishedAt := h.DOM.ChildrenFiltered("h4").Text()
		t, err := time.Parse("January 02, 2006", strings.TrimPrefix(publishedAt, "Published "))
		if err != nil {
			return
		}

		postBody, err := h.DOM.Find("#post-content").Html()
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

	err := c.listCollector.Request("GET", "https://www.micahlerner.com/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	c.postCollector.Wait()
	return nil
}
