// Package collectors implement all collector.
package collectors

import (
	"context"
	"time"
)

// Post is an blog posts.
type Post struct {
	Domain      string
	Title       string
	Path        string
	PublishedAt time.Time
	Content     string
}

// Collector is used to collect article from each domain.
type Collector interface {
	Name() string
	Initialize(ctx context.Context) error
	Start(ctx context.Context, ch chan<- Post) error
}
