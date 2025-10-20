package app

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
)

func TestNewApplication(t *testing.T) {
	g := gomega.NewWithT(t)
	a, err := NewApplication(context.Background())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(a).ToNot(gomega.BeNil())
	g.Expect(a.summaryAgent).ToNot(gomega.BeNil())
	g.Expect(a.store).ToNot(gomega.BeNil())
	g.Expect(a.f).ToNot(gomega.BeNil())
}
