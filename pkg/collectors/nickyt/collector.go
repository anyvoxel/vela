// Package nickyt implement collector for https://www.nickyt.co/archive/
package nickyt

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
		"vela.collectors.nickyt",
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
	return "nickyt"
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

	postCollector.OnHTML("#main-content", func(h *colly.HTMLElement) {
		var postBody string
		postBody, err = h.DOM.Find("article").Html()
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

	c.listCollector.OnHTML("section.post-list", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("li.post-list__item", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("h3 > a.post-list__link", "href")
			if path == "" {
				return true
			}
			path = "https://www.nickyt.co" + path

			title := h.ChildText("h3 > a.post-list__link")
			publishedAt := h.ChildAttr("p > time", "datetime")
			t, err := time.Parse(time.RFC3339, publishedAt)
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

	err := c.listCollector.Request("GET", "https://www.nickyt.co/archive/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
