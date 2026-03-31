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
	s := &summarizerImpl{
		summarizeType: "image",
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

func TestNormalizeResolveBaseURL(t *testing.T) {
	g := gomega.NewWithT(t)

	got, err := normalizeResolveBaseURL("https://example.com/blog/index.html?x=1#frag")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(got).To(gomega.Equal("https://example.com/"))
}

func TestBuildPostsResolveRelativeURLWithOrigin(t *testing.T) {
	g := gomega.NewWithT(t)

	result := listParseResult{
		Items: []listParseItem{
			{URL: "posts/1", Title: "t1"},
			{URL: "/posts/2", Title: "t2"},
			{URL: "https://example.com/posts/3", Title: "t3"},
		},
	}

	posts, err := buildPosts(result, "https://example.com/blog/index.html", "example")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(posts).To(gomega.HaveLen(3))
	g.Expect(posts[0].Path).To(gomega.Equal("https://example.com/posts/1"))
	g.Expect(posts[1].Path).To(gomega.Equal("https://example.com/posts/2"))
	g.Expect(posts[2].Path).To(gomega.Equal("https://example.com/posts/3"))
}

func TestTruncateForLog_UTF8Safe(t *testing.T) {
	g := gomega.NewWithT(t)

	// 3 bytes per Chinese rune in UTF-8
	s := "你好世界abc"

	got := truncateForLog(s, 4)
	g.Expect(got).To(gomega.Equal("你...(truncated)"))

	got = truncateForLog(s, 6)
	g.Expect(got).To(gomega.Equal("你好...(truncated)"))

	got = truncateForLog(s, 9)
	g.Expect(got).To(gomega.Equal("你好世...(truncated)"))
}
