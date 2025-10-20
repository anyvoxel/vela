// Package storage contain all implementation for posts storage.
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
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	PublishedAt time.Time `json:"published_at"`
}

func openFile(_ context.Context) (*os.File, error) {
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

// Write all result to current file
func Write(ctx context.Context, results []*SummaryResult) error {
	if len(results) == 0 {
		return nil
	}

	monthStr := time.Now().UTC().Format("200601")
	_, err := os.Stat(path.Join("./data", monthStr))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		err = os.MkdirAll(path.Join("./data", monthStr), 0755)
		if err != nil {
			return err
		}
	}

	f, err := openFile(ctx)
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
