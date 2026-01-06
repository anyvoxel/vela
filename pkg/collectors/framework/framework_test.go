package framework

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/anyvoxel/vela/pkg/collectors"
	"github.com/anyvoxel/vela/pkg/collectors/mocks"
)

func TestFramework_AfterPropertiesSet_DuplicateCollectorName(t *testing.T) {
	g := gomega.NewWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c1 := mocks.NewMockCollector(mockCtrl)
	c1.EXPECT().Name().Return("collector-a").AnyTimes()
	c2 := mocks.NewMockCollector(mockCtrl)
	c2.EXPECT().Name().Return("collector-b").AnyTimes()
	c3 := mocks.NewMockCollector(mockCtrl)
	c3.EXPECT().Name().Return("collector-a").AnyTimes()

	// Create a Framework with duplicate collectors
	framework := &Framework{
		cs: []collectors.Collector{
			c1,
			c2,
			c3,
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c1 := mocks.NewMockCollector(mockCtrl)
	c1.EXPECT().Name().Return("collector-a").AnyTimes()
	c2 := mocks.NewMockCollector(mockCtrl)
	c2.EXPECT().Name().Return("collector-b").AnyTimes()
	c3 := mocks.NewMockCollector(mockCtrl)
	c3.EXPECT().Name().Return("collector-c").AnyTimes()

	// Create a Framework with unique collectors
	framework := &Framework{
		cs: []collectors.Collector{
			c1,
			c2,
			c3,
		},
	}

	// Call the AfterPropertiesSet method
	err := framework.AfterPropertiesSet(context.Background())

	// Assert that no error is returned
	g.Expect(err).NotTo(gomega.HaveOccurred())
}
