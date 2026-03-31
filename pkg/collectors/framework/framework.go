// Package framework contains all collector implementation.
package framework

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/anyvoxel/airmid/anvil"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"
	slogctx "github.com/veqryn/slog-context"

	"github.com/anyvoxel/vela/pkg/apitypes"
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

var (
	_ ioc.InitializingBean = (*Framework)(nil)

	// ErrDuplicateCollector is returned when a collector with the same name already exists.
	ErrDuplicateCollector    = errors.New("duplicate collector name")
	errListParserUnavailable = errors.New("collector sources configured but listParser is not available")
)

// Framework will orchestration all collectors.
type Framework struct {
	cs []collectors.Collector `airmid:"autowire:?"`

	// sourcesFile is a path to a JSON file that contains an array of CollectorSource.
	// Example file content:
	//  [{"name":"example","url":"https://example.com/archive","headers":{"User-Agent":"..."}}]
	sourcesFile string `airmid:"value:${vela.collectors.sources_file:=./collectors.json}"`

	listParser collectors.ListParser `airmid:"autowire:?"`
}

// NewFramework creates a new Framework with the given collectors.
// This is intended for testing purposes.
func NewFramework(cs []collectors.Collector) *Framework {
	return &Framework{cs: cs}
}

// AfterPropertiesSet implements ioc.InitializingBean.
func (f *Framework) AfterPropertiesSet(_ context.Context) error {
	if err := f.appendConfiguredCollectors(); err != nil {
		return err
	}
	return f.ensureUniqueCollectorNames()
}

func (f *Framework) appendConfiguredCollectors() error {
	filePath := strings.TrimSpace(f.sourcesFile)
	if filePath == "" {
		return nil
	}
	if f.listParser == nil {
		return errListParserUnavailable
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read vela.collectors.sources_file %q failed: %w", filePath, err)
	}

	var sources []CollectorSource
	if err := json.Unmarshal(b, &sources); err != nil {
		return fmt.Errorf("invalid sources file %q: %w", filePath, err)
	}
	for i, src := range sources {
		cc, err := newConfiguredCollector(src, f.listParser)
		if err != nil {
			return fmt.Errorf("invalid sources file %q item[%d]: %w", filePath, i, err)
		}
		f.cs = append(f.cs, cc)
	}
	return nil
}

func (f *Framework) ensureUniqueCollectorNames() error {
	// ensure all collector names are unique
	names := make(map[string]struct{})
	for _, c := range f.cs {
		if _, ok := names[c.Name()]; ok {
			return fmt.Errorf("%w: %s", ErrDuplicateCollector, c.Name())
		}
		names[c.Name()] = struct{}{}
	}
	return nil
}

// Start will collector post from all domain.
func (f *Framework) Start(ctx context.Context, ch chan<- apitypes.Post) error {
	slogctx.FromCtx(ctx).InfoContext(ctx, "start to process collector",
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
		cch := make(chan apitypes.Post, 10)

		go func(c collectors.Collector) {
			defer wg.Done()
			defer close(cch)

			err := c.Start(
				slogctx.With(ctx, slog.String("Collector", c.Name())),
				cch)
			if err != nil {
				slogctx.FromCtx(ctx).ErrorContext(ctx, "start collector failed",
					slog.String("Collector", c.Name()),
					slog.Any("Error", err),
				)
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
