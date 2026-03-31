// Package apitypes contains common type
package apitypes

import (
	"time"
)

// Post is an blog posts.
type Post struct {
	// It's the collector's name
	Domain string

	Title string

	// It's the full URL path of current post
	Path        string
	PublishedAt time.Time
}
