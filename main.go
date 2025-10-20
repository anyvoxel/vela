// Package main ...
package main

import (
	"context"
	"log/slog"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/anyvoxel/vela/pkg/agents"
	"github.com/anyvoxel/vela/pkg/storage"
	"github.com/gocolly/colly/v2"
)

func main() { //nolint
	summaryAgent, err := agents.NewSummarizer(context.Background())
	if err != nil {
		panic(err)
	}

	listCollector := colly.NewCollector()
	articleCollector := colly.NewCollector()

	results := make([]*storage.SummaryResult, 0)

	listCollector.OnHTML("div.Blog", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("div.blog-posts div.post", func(_ int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("h3.post-title a", "href")
			if path == "" {
				return false
			}

			slog.InfoContext(context.Background(), "collect article", slog.String("Path", path))
			err := articleCollector.Request("GET", path, nil, h.Request.Ctx, nil)
			if err != nil {
				slog.ErrorContext(context.TODO(),
					"cann't request on article",
					slog.Any("Error", err),
					slog.String("NextPath", path))
				return false
			}

			return true
		})
	})

	articleCollector.OnHTML("div.post", func(h *colly.HTMLElement) {
		title := h.ChildText("h3.post-title")
		if title == "" {
			return
		}

		publishedAt := h.ChildAttr("time.published", "datetime")
		t, err := time.Parse("2006-01-02T15:04:05-07:00", publishedAt)
		if err != nil {
			return
		}

		postBody, err := h.DOM.Find("div.post-body").Html()
		if err != nil {
			return
		}
		bodyMarkdown, err := md.NewConverter("", true, nil).ConvertString(postBody)
		if err != nil {
			return
		}

		result, err := summaryAgent.Summary(context.Background(), bodyMarkdown)
		if err != nil {
			panic(err)
		}
		results = append(results, &storage.SummaryResult{
			Title:       title,
			Summary:     result,
			PublishedAt: t.UTC(),
		})
	})

	err = listCollector.Request("GET", "http://muratbuffalo.blogspot.com", nil, colly.NewContext(), nil)
	if err != nil {
		panic(err)
	}

	listCollector.Wait()
	articleCollector.Wait()

	err = storage.Write(context.Background(), results)
	if err != nil {
		panic(err)
	}
	slog.InfoContext(context.Background(), "process done")
}
