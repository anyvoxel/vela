package agents

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

func TestNewSummarizer(t *testing.T) {
	g := gomega.NewWithT(t)
	s := &Summarizer{
		summarizeType: "text",
	}
	err := s.AfterPropertiesSet(context.Background())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(s).ToNot(gomega.BeNil())
	g.Expect(s.systemPrompt).ToNot(gomega.Equal(""))

	publishedAt := "(5/8/2025)"
	timeAt, err := time.Parse("1/2/2006", strings.TrimLeft(strings.TrimRight(publishedAt, ")"), "("))
	g.Expect(timeAt).To(gomega.Equal(time.Date(2025, 5, 8, 0, 0, 0, 0, time.UTC)))
	g.Expect(err).ToNot(gomega.HaveOccurred())
	timeAt, err = time.Parse("(1/2/2006)", publishedAt)
	g.Expect(timeAt).To(gomega.Equal(time.Date(2025, 5, 8, 0, 0, 0, 0, time.UTC)))
	g.Expect(err).ToNot(gomega.HaveOccurred())
}
