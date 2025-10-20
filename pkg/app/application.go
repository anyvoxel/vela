// Package app will take care of all logic.
package app

import (
	"context"
	"log/slog"
	"sync"

	"github.com/anyvoxel/vela/pkg/agents"
	"github.com/anyvoxel/vela/pkg/collectors"
	"github.com/anyvoxel/vela/pkg/collectors/framework"
	"github.com/anyvoxel/vela/pkg/storage"
)

// Application is the represent of vela.
type Application struct {
	f            *framework.Framework
	summaryAgent *agents.Summarizer
	store        *storage.Storage
}

// NewApplication will creates an app implementation.
func NewApplication(ctx context.Context) (*Application, error) {
	summaryAgent, err := agents.NewSummarizer(ctx)
	if err != nil {
		return nil, err
	}

	store, err := storage.NewStorage(ctx, "./")
	if err != nil {
		return nil, err
	}

	f, err := framework.NewFramework(ctx)
	if err != nil {
		return nil, err
	}

	return &Application{
		f:            f,
		summaryAgent: summaryAgent,
		store:        store,
	}, nil
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
