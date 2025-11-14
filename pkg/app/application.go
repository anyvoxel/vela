// Package app will take care of all logic.
package app

import (
	"context"
	"log/slog"
	"reflect"
	"sync"

	"github.com/anyvoxel/airmid/anvil"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"

	"github.com/anyvoxel/vela/pkg/agents"
	"github.com/anyvoxel/vela/pkg/collectors"
	"github.com/anyvoxel/vela/pkg/collectors/framework"
	"github.com/anyvoxel/vela/pkg/storage"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.application",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Application)(nil)),
		),
	))
}

// Application is the represent of vela.
type Application struct {
	f            *framework.Framework `airmid:"autowire:?"`
	summaryAgent *agents.Summarizer   `airmid:"autowire:?"`
	store        *storage.Storage     `airmid:"autowire:?"`

	airmidApplication airapp.Application
}

var (
	_ airapp.Runner           = (*Application)(nil)
	_ airapp.ApplicationAware = (*Application)(nil)
)

// Run implement Runner.Run
func (a *Application) Run(ctx context.Context) {
	go func() {
		err := a.Start(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "start application failed", slog.Any("Error", err))
		}

		a.airmidApplication.Shutdown()
	}()
}

// Stop implement Runner.Stop
func (a *Application) Stop(_ context.Context) {}

// SetApplication implement ApplicationAware
func (a *Application) SetApplication(application airapp.Application) {
	a.airmidApplication = application
}

// Start will start the application
func (a *Application) Start(ctx context.Context) error {
	ch := make(chan collectors.Post, 100)
	results := make([]*storage.SummaryResult, 0)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := a.f.Start(ctx, ch)
		if err != nil {
			slog.ErrorContext(ctx, "start framework failed")
		}
	}()

	go func() {
		defer wg.Done()

		for post := range ch {
			if a.store.SummaryExists(ctx, post.Domain, post.Title) {
				continue
			}

			result, err := a.summaryAgent.Summary(ctx, post.Content)
			if err != nil {
				slog.ErrorContext(ctx,
					"summary post failed",
					slog.String("Path", post.Path),
					slog.String("Domain", post.Domain),
					slog.String("Title", post.Title),
					slog.Any("Error", err),
				)
				continue
			}

			results = append(results, &storage.SummaryResult{
				Domain:      post.Domain,
				Path:        post.Path,
				Title:       post.Title,
				Summary:     result,
				PublishedAt: post.PublishedAt,
			})
		}
	}()

	wg.Wait()

	err := a.store.Put(ctx, results)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "process done")
	return nil
}
