// Package storage implement file system jsonl storage.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
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
	existPosts map[string]map[string]bool
	dataPath   string
}

// NewStorage creates a file system implementation.
func NewStorage(ctx context.Context, dir string) (*Storage, error) {
	dataPath := path.Join(dir, "data")
	_, err := os.Stat(dataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		err = os.MkdirAll(dataPath, 0755)
		if err != nil {
			return nil, err
		}
	}

	s := &Storage{
		existPosts: map[string]map[string]bool{},
		dataPath:   dataPath,
	}

	err = s.readPreviousSummary(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) readPreviousSummary(ctx context.Context) error {
	dirEntries, err := os.ReadDir(s.dataPath)
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}

		err = s.readPreviousSummarySubDir(ctx, dirEntry.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) readPreviousSummarySubDir(ctx context.Context, subdir string) error {
	dirEntries, err := os.ReadDir(path.Join(s.dataPath, subdir))
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		if !strings.HasSuffix(dirEntry.Name(), ".jsonl") {
			continue
		}

		err = s.readPreviousSummaryFile(ctx, path.Join(s.dataPath, subdir), dirEntry.Name())
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) readPreviousSummaryFile(_ context.Context, dir string, file string) error {
	f, err := os.Open(path.Join(dir, file))
	if err != nil {
		return err
	}
	defer f.Close() //nolint

	decoder := json.NewDecoder(f)
	for {
		var result SummaryResult
		err := decoder.Decode(&result)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		domain, ok := s.existPosts[result.Domain]
		if !ok {
			s.existPosts[result.Domain] = map[string]bool{
				result.Title: true,
			}
			continue
		}

		domain[result.Title] = true
	}
}

func (s *Storage) summaryExists(_ context.Context, result *SummaryResult) bool {
	domain, ok := s.existPosts[result.Domain]
	if !ok {
		return false
	}

	return domain[result.Title]
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
		if s.summaryExists(ctx, result) {
			continue
		}

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
