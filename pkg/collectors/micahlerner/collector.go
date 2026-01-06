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
	"github.com/clbanning/mxj/v2"
	"github.com/gocolly/colly/v2"
	slogctx "github.com/veqryn/slog-context"

	"github.com/anyvoxel/vela/pkg/apitypes"
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
	return "micahlerner"
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

	postCollector.OnHTML("body article", func(h *colly.HTMLElement) {
		var postBody string
		postBody, err = h.DOM.Find("#post-content").Html()
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

func (c *Collector) extracePublishedAt(ctx context.Context, h *colly.HTMLElement) time.Time {
	body, err := h.DOM.Html()
	if err != nil {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"convert DOM to Html failed",
			slog.Any("Error", err),
		)
		return time.Time{}
	}

	m, err := mxj.NewMapXml([]byte(body))
	if err != nil {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"convert Html to Map failed",
			slog.Any("Error", err),
		)
		return time.Time{}
	}

	obj, ok := m["p"]
	if !ok {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"Html didn't have 'p' element",
		)
		return time.Time{}
	}
	objv, ok := obj.(map[string]interface{})
	if !ok {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"'p' element is invalid type",
			slog.Any("Obj", obj),
		)
		return time.Time{}
	}

	obj, ok = objv["#text"]
	if !ok {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"'p' element didn't have '#text' element",
		)
		return time.Time{}
	}

	publishedAt, ok := obj.(string)
	if !ok {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"'#text' element is invalid type",
			slog.Any("Obj", obj),
		)
		return time.Time{}
	}

	publishedAt = strings.ReplaceAll(publishedAt, "\n", "")
	t, err := time.Parse("- January 02, 2006", strings.TrimSpace(publishedAt))
	if err != nil {
		slogctx.FromCtx(ctx).ErrorContext(ctx,
			"parse datetime failed",
			slog.Any("Error", err),
			slog.String("PublishedAt", publishedAt),
		)
		return time.Time{}
	}
	return t
}

// Start implement collector.Start
func (c *Collector) Start(ctx context.Context, ch chan<- apitypes.Post) error {

	c.listCollector.OnHTML("body article section div ul", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("li", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("p a", "href")
			if path == "" {
				return true
			}
			path = "https://www.micahlerner.com" + path

			title := h.ChildText("p a")
			t := c.extracePublishedAt(slogctx.With(ctx, slog.String("Path", path)), h)

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

	err := c.listCollector.Request("GET", "https://www.micahlerner.com/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
