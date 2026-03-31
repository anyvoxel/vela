// Package collectors implement all collector.
package collectors

import (
	"context"

	"github.com/anyvoxel/vela/pkg/apitypes"
)

// Collector is used to collect article from each domain.
type Collector interface {
	// Name return unique collector name
	Name() string

	// Initialize to prepare anything before Start
	Initialize(ctx context.Context) error

	// Start will collector post from this domain
	Start(ctx context.Context, ch chan<- apitypes.Post) error
}

// ListParser extracts post metadata from a list page.
// It should return absolute URLs in Post.Path.
type ListParser interface {
	ParseList(ctx context.Context, html, baseURL, domain string) ([]apitypes.Post, error)
}
