// Package framework contains all collector implementation.
package framework

import (
	"context"
	"log/slog"
	"sync"

	"github.com/anyvoxel/vela/pkg/collectors"
	"github.com/anyvoxel/vela/pkg/collectors/allthingsdistributed"
	"github.com/anyvoxel/vela/pkg/collectors/micahlerner"
	"github.com/anyvoxel/vela/pkg/collectors/muratbuffalo"
)

// Framework will orchestration all collectors.
type Framework struct {
	cs []collectors.Collector
}

type newFunc func(context.Context) (collectors.Collector, error)

// NewFramework creates an framework implementation.
func NewFramework(ctx context.Context) (*Framework, error) {
	newFuncs := []newFunc{
		muratbuffalo.NewCollector,
		allthingsdistributed.NewCollector,
		micahlerner.NewCollector,
	}
	f := &Framework{
		cs: make([]collectors.Collector, 0, len(newFuncs)),
	}
	for _, fn := range newFuncs {
		c, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		f.cs = append(f.cs, c)
	}

	return f, nil
}

// Start will collector post from all domain.
func (f *Framework) Start(ctx context.Context, ch chan<- collectors.Post) error {
	defer close(ch)

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
