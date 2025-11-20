package agents

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
)

func TestNewSummarizer(t *testing.T) {
	g := gomega.NewWithT(t)
	s := &Summarizer{}
	err := s.AfterPropertiesSet(context.Background())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(s).ToNot(gomega.BeNil())
	g.Expect(s.systemPrompt).ToNot(gomega.Equal(""))
}
