// Package engineeringfb implement collector for https://engineering.fb.com/
package engineeringfb

import (
	"context"
	"log/slog"
	"net/url"
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
		"vela.collectors.engineeringfb",
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
	return "engineeringfb"
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

	postCollector.OnHTML("article.post", func(h *colly.HTMLElement) {
		var postBody string
		postBody, err = h.DOM.Find("div.entry-content").Html()
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

func (c *Collector) extracePublishedAt(ctx context.Context, h *colly.HTMLElement, path string) time.Time {
	publishedAt := h.ChildText("header.entry-header span.posted-on time.published")
	if publishedAt != "" {
		t, err := time.Parse("Jan 02, 2006", publishedAt)
		if err != nil {
			slogctx.FromCtx(ctx).ErrorContext(ctx,
				"parse datetime failed",
				slog.Any("Error", err),
				slog.String("Path", path))
			return time.Time{}
		}

		return t
	}

	// If entry-header didn't have PublishedAt, we try to extrace it from path
	urlPath, err := url.Parse(path)
	if err != nil {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"parse url path failed",
			slog.Any("Error", err),
			slog.String("Path", path))
		return time.Time{}
	}

	if len(urlPath.Path) < 11 {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"can't parse datetime from url path, it's two short",
			slog.Any("Error", err),
			slog.String("Path", path))
		return time.Time{}
	}

	t, err := time.Parse("/2006/01/02", urlPath.Path[:11])
	if err != nil {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"parse datetime from url path failed",
			slog.Any("Error", err),
			slog.String("Path", path))
		return time.Time{}
	}
	return t
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- apitypes.Post) error {

	c.listCollector.OnHTML("#primary", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("article.post", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("div.entry-title a", "href")
			if path == "" {
				return true
			}

			title := h.ChildText("div.entry-title a")
			publishedAt := c.extracePublishedAt(ctx, h, path)
			slogctx.FromCtx(ctx).InfoContext(ctx, "collect article",
				slog.String("Path", path),
				slog.String("Title", title),
				slog.Any("PublishedAt", publishedAt),
			)
			post := apitypes.Post{
				Domain:      c.Name(),
				Title:       title,
				Path:        path,
				PublishedAt: publishedAt,
			}
			ch <- post

			return true
		})
	})

	err := c.listCollector.Request("GET", "https://engineering.fb.com/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
