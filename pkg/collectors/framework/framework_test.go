package framework

import (
	"context"
	"testing"

	"github.com/onsi/gomega"

	"github.com/anyvoxel/vela/pkg/apitypes"
	"github.com/anyvoxel/vela/pkg/collectors"
)

// MockCollector is a mock implementation of the Collector interface for testing.
type MockCollector struct {
	name string
}

func (m *MockCollector) Name() string {
	return m.name
}

func (m *MockCollector) Initialize(_ context.Context) error {
	return nil
}

func (m *MockCollector) Start(_ context.Context, _ chan<- apitypes.Post) error {
	return nil
}

func (m *MockCollector) ResolvePostContent(_ context.Context, _ apitypes.Post) (string, error) {
	return "", nil
}

func TestFramework_AfterPropertiesSet_DuplicateCollectorName(t *testing.T) {
	g := gomega.NewWithT(t)
	// Create a Framework with duplicate collectors
	framework := &Framework{
		cs: []collectors.Collector{
			&MockCollector{name: "collector-a"},
			&MockCollector{name: "collector-b"},
			&MockCollector{name: "collector-a"},
		},
	}

	// Call the AfterPropertiesSet method
	err := framework.AfterPropertiesSet(context.Background())

	// Assert that an error is returned
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err.Error()).To(gomega.Equal("duplicate collector name: collector-a"))
}

func TestFramework_AfterPropertiesSet_NoDuplicateCollectorName(t *testing.T) {
	g := gomega.NewWithT(t)
	// Create a Framework with unique collectors
	framework := &Framework{
		cs: []collectors.Collector{
			&MockCollector{name: "collector-a"},
			&MockCollector{name: "collector-b"},
			&MockCollector{name: "collector-c"},
		},
	}

	// Call the AfterPropertiesSet method
	err := framework.AfterPropertiesSet(context.Background())

	// Assert that no error is returned
	g.Expect(err).NotTo(gomega.HaveOccurred())
}
