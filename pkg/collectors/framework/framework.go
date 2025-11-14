// Package framework contains all collector implementation.
package framework

import (
	"context"
	"log/slog"
	"reflect"
	"sync"

	"github.com/anyvoxel/airmid/anvil"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"

	"github.com/anyvoxel/vela/pkg/collectors"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.collectors.framework",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Framework)(nil)),
		),
	))
}

// Framework will orchestration all collectors.
type Framework struct {
	cs []collectors.Collector `airmid:"autowire:?"`
}

// Start will collector post from all domain.
func (f *Framework) Start(ctx context.Context, ch chan<- collectors.Post) error {
	defer close(ch)

	slog.InfoContext(ctx, "start to process collector",
		slog.Int("CollectorCount", len(f.cs)),
	)

	for _, c := range f.cs {
		err := c.Initialize(ctx)
		if err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	for _, c := range f.cs {
		wg.Add(2)
		cch := make(chan collectors.Post, 10)

		go func(c collectors.Collector) {
			defer wg.Done()

			err := c.Start(ctx, cch)
			if err != nil {
				slog.ErrorContext(ctx, "start collector failed")
			}
		}(c)

		go func(c collectors.Collector) {
			defer wg.Done()

			for post := range cch {
				post.Domain = c.Name()
				ch <- post
			}
		}(c)
	}

	wg.Wait()
	return nil
}
