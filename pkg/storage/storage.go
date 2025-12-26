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
	"reflect"
	"strings"
	"time"

	"github.com/anyvoxel/airmid/anvil"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"
	slogctx "github.com/veqryn/slog-context"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.storage.storage",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Storage)(nil)),
		),
	))
}

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
	existPosts map[string]bool
	dataPath   string
	dir        string `airmid:"value:${vela.storage.dir:=./}"`
}

var (
	_ ioc.InitializingBean = (*Storage)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (s *Storage) AfterPropertiesSet(ctx context.Context) error {
	dataPath := path.Join(s.dir, "data")
	_, err := os.Stat(dataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		err = os.MkdirAll(dataPath, 0755)
		if err != nil {
			return err
		}
	}

	s.existPosts = map[string]bool{}
	s.dataPath = dataPath

	err = s.readPreviousSummary(ctx)
	if err != nil {
		return err
	}

	return nil
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

func (s *Storage) readPreviousSummaryFile(ctx context.Context, dir string, file string) error {
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

		_, ok := s.existPosts[result.Path]
		if ok {
			slogctx.FromCtx(ctx).ErrorContext(ctx,
				"duplicate path in storage",
				slog.String("Path", result.Path),
			)
		}
		s.existPosts[result.Path] = true

		if len(strings.Split(result.Title, "\n")) > 1 {
			slogctx.FromCtx(ctx).ErrorContext(ctx,
				"post title has multi line",
				slog.String("Path", result.Path),
			)
		}
	}
}

// SummaryExists return true if this summary already persist
func (s *Storage) SummaryExists(_ context.Context, path string) bool {
	return s.existPosts[path]
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
		if s.SummaryExists(ctx, result.Path) {
			continue
		}

		err = json.NewEncoder(f).Encode(result)
		if err != nil {
			return err
		}
	}
	slogctx.FromCtx(ctx).InfoContext(ctx, "save results",
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
