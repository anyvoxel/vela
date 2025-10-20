// Package storage implement file system jsonl storage.
package storage

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path"
	"time"
)

// SummaryResult is the result of a summary.
type SummaryResult struct {
	Domain      string    `json:"domain"`
	Path        string    `json:"path"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	PublishedAt time.Time `json:"published_at"`
}

// Storage will access and persist to all previous posts.
type Storage struct {
}

// NewStorage creates a file system implementation.
func NewStorage(_ context.Context, dir string) (*Storage, error) {
	_, err := os.Stat(path.Join(dir, "data"))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		err = os.MkdirAll(path.Join(dir, "data"), 0755)
		if err != nil {
			return nil, err
		}
	}

	return &Storage{}, nil
}

// Put will persist all result to jsonl file.
func (s *Storage) Put(ctx context.Context, results []*SummaryResult) error {
	if len(results) == 0 {
		return nil
	}

	f, err := s.openFile(ctx)
	if err != nil {
		return err
	}
	defer f.Close() //nolint

	for _, result := range results {
		err = json.NewEncoder(f).Encode(result)
		if err != nil {
			return err
		}
	}
	slog.InfoContext(ctx, "save results",
		slog.String("Filename", f.Name()),
		slog.Int("Rows", len(results)))
	return nil
}

func (s *Storage) openFile(_ context.Context) (*os.File, error) {
	monthStr := time.Now().UTC().Format("200601")
	_, err := os.Stat(path.Join("./data", monthStr))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		err = os.MkdirAll(path.Join("./data", monthStr), 0755)
		if err != nil {
			return nil, err
		}
	}

	dataStr := time.Now().UTC().Format("20060102")
	filename := path.Join("./data", monthStr, dataStr+".jsonl")
	_, err = os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		f, err := os.Create(filename)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}
