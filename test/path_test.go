package test

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestCurrentProjectPath(t *testing.T) {
	t.Run("normal test", func(t *testing.T) {
		g := gomega.NewWithT(t)

		path := CurrentProjectPath()

		// NOTE: the '/code' path is used with code pipeline.
		// When code running in the pipeline, the codebase will copy to /home/code directory.
		g.Expect(path).To(gomega.MatchRegexp("(/vela$)|(/code$)"))
	})
}
