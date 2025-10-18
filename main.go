// Package main ...
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-kratos/blades"
	bladesopenai "github.com/go-kratos/blades/contrib/openai"
	"github.com/gocolly/colly/v2"
)

func main() {
	agent := blades.NewAgent("Template Agent",
		blades.WithModel(os.Getenv("OPENAI_MODEL")), blades.WithProvider(bladesopenai.NewChatProvider()))

	listCollector := colly.NewCollector()
	articleCollector := colly.NewCollector()

	listCollector.OnHTML("div.Blog", func(h *colly.HTMLElement) {
		shouldNextPage := true

		h.ForEachWithBreak("div.blog-posts div.post", func(i int, h *colly.HTMLElement) bool {
			path := h.ChildAttr("h3.post-title a", "href")
			if path == "" {
				return false
			}

			fmt.Printf("process article: %s\n", path)
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

		if !shouldNextPage {
			return
		}

		path := h.ChildAttr("div.blog-pager a.blog-pager-older-link", "href")
		if path == "" {
			return
		}

		err := listCollector.Request("GET", path, nil, h.Request.Ctx, nil)
		if err != nil {
			return
		}
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

		fmt.Printf("time: %v\n", t.UTC())

		params := map[string]any{
			"context": bodyMarkdown,
		}
		prompt, err := blades.NewPromptTemplate().System(
			"请总结文档 {{.content}} 的主要内容, 输出应当尽量简短， 控制在 100 字内。", params).User("", params).Build()
		if err != nil {
			return
		}

		result, err := agent.Run(context.Background(), prompt)
		if err != nil {
			panic(err)
		}
		fmt.Printf("title: %s, summary: %s\n", title, result.Text())
	})

	err := listCollector.Request("GET", "http://muratbuffalo.blogspot.com", nil, colly.NewContext(), nil)
	if err != nil {
		panic(err)
	}

	listCollector.Wait()
	articleCollector.Wait()

	println("Hello, World!")
}
