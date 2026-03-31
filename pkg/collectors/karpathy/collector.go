// Package karpathy implement collector for https://karpathy.bearblog.dev/blog/
package karpathy

import (
	"context"
	"log/slog"
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
		"vela.collectors.karpathy",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Collector)(nil)),
		),
	))
}

// Collector implemetation.
type Collector struct {
	listCollector *colly.Collector
	listParser    collectors.ListParser `airmid:"autowire:?"`
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
	return "karpathy"
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

	err := c.listCollector.Request("GET", "https://karpathy.bearblog.dev/blog/", nil, colly.NewContext(), nil)
	if err != nil {
		return err
	}

	c.listCollector.Wait()
	return nil
}
