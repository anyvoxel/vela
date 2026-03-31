// Package uberblog implement collector for https://www.uber.com/en-SG/blog/
package uberblog

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"

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
		"vela.collectors.uberblog",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Collector)(nil)),
		),
	))
}

// Collector implemetation.
type Collector struct {
	listCollector *colly.Collector
	header        http.Header
	listParser    collectors.ListParser `airmid:"autowire:?"`
}

var (
	_ ioc.InitializingBean = (*Collector)(nil)
	_ collectors.Collector = (*Collector)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (c *Collector) AfterPropertiesSet(context.Context) error {
	header := http.Header{}
	header.Add("Content-Type", "text/html; charset=utf-8")
	header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7") //nolint
	header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")               //nolint

	c.listCollector = colly.NewCollector()
	c.header = header
	return nil
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
func (c *Collector) Start(ctx context.Context, ch chan<- apitypes.Post) error {
	c.listCollector.OnResponse(func(r *colly.Response) {
		posts, err := c.listParser.ParseList(ctx, string(r.Body), r.Request.URL.String(), c.Name())
		if err != nil {
			slogctx.FromCtx(ctx).ErrorContext(ctx,
				"parse list failed",
				slog.Any("Error", err),
				slog.String("URL", r.Request.URL.String()),
			)
			return
		}
		for _, post := range posts {
			slogctx.FromCtx(ctx).InfoContext(ctx, "collect article",
				slog.String("Path", post.Path),
				slog.String("Title", post.Title),
				slog.Any("PublishedAt", post.PublishedAt),
			)
			ch <- post
		}
	})

	err := c.listCollector.Request("GET", "https://www.uber.com/en-SG/blog/", nil, colly.NewContext(), c.header)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
