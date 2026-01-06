package app

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/anyvoxel/vela/pkg/agents"
	"github.com/anyvoxel/vela/pkg/apitypes"
	"github.com/anyvoxel/vela/pkg/collectors"
	"github.com/anyvoxel/vela/pkg/collectors/framework"
	"github.com/anyvoxel/vela/pkg/storage"
)

type testCollector struct {
	startFunc func(ctx context.Context, ch chan<- apitypes.Post) error
}

func (t *testCollector) Name() string {
	return "test-collector"
}

func (t *testCollector) Initialize(_ context.Context) error {
	return nil
}

func (t *testCollector) Start(ctx context.Context, ch chan<- apitypes.Post) error {
	if t.startFunc != nil {
		return t.startFunc(ctx, ch)
	}
	return nil
}

func (t *testCollector) ResolvePostContent(_ context.Context, _ apitypes.Post) (string, error) {
	return "content", nil
}

func TestApplication_Start_ChannelClosing(t *testing.T) {
	g := gomega.NewWithT(t)

	var wg sync.WaitGroup
	wg.Add(1)

	collector := &testCollector{
		startFunc: func(_ context.Context, ch chan<- apitypes.Post) error {
			ch <- apitypes.Post{Title: "post1", Path: "/post1"}
			return nil
		},
	}

	f := framework.NewFramework([]collectors.Collector{collector})

	// Create a temporary directory for storage
	tempDir, err := os.MkdirTemp("", "vela-test-")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	defer os.RemoveAll(tempDir)

	s := storage.NewStorage(tempDir)
	err = s.AfterPropertiesSet(context.Background())
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Create a summarizer and mock the summary function
	summarizer := &agents.Summarizer{}
	summarizer.Summary = func(_ context.Context, _ apitypes.Post) (string, error) {
		return "summary", nil
	}

	originalPut := s.Put
	s.Put = func(ctx context.Context, results []*storage.SummaryResult) error {
		defer wg.Done()
		return originalPut(ctx, results)
	}

	app := &Application{
		f:            f,
		store:        s,
		summaryAgent: summarizer,
	}

	done := make(chan struct{})
	go func() {
		err := app.Start(context.Background())
		g.Expect(err).ToNot(gomega.HaveOccurred())
		close(done)
	}()

	select {
	case <-done:
		// test passed
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out. The channel was not closed.")
	}

	wg.Wait()
}
