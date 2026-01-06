package app

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	mock_agents "github.com/anyvoxel/vela/pkg/agents/mocks"
	"github.com/anyvoxel/vela/pkg/apitypes"
	"github.com/anyvoxel/vela/pkg/collectors"
	"github.com/anyvoxel/vela/pkg/collectors/framework"
	mock_collectors "github.com/anyvoxel/vela/pkg/collectors/mocks"
	mock_storage "github.com/anyvoxel/vela/pkg/storage/mocks"
)

func TestApplication_Start_ChannelClosing(t *testing.T) {
	g := gomega.NewWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCollector := mock_collectors.NewMockCollector(mockCtrl)
	mockCollector.EXPECT().Name().Return("test-collector").AnyTimes()
	mockCollector.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
	mockCollector.EXPECT().Start(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ch chan<- apitypes.Post) error {
			ch <- apitypes.Post{Title: "post1", Path: "/post1"}
			return nil
		}).AnyTimes()
	mockCollector.EXPECT().ResolvePostContent(gomock.Any(), gomock.Any()).Return("content", nil).AnyTimes()

	f := framework.NewFramework([]collectors.Collector{mockCollector})

	s := mock_storage.NewMockStorage(mockCtrl)
	s.EXPECT().SummaryExists(gomock.Any(), "/post1").Return(false)
	s.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

	// Create a summarizer and mock the summary function
	summarizer := mock_agents.NewMockSummarizer(mockCtrl)
	summarizer.EXPECT().Summary(gomock.Any(), gomock.Any()).Return("summary", nil)

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
}
