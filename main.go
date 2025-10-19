// Package main ...
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-kratos/blades"
	bladesopenai "github.com/go-kratos/blades/contrib/openai"
	"github.com/gocolly/colly/v2"
)

// SummaryAgent is a agent that can summarize a blog post.
type SummaryAgent struct {
	agent        *blades.Agent
	systemPrompt string
}

// NewSummaryAgent creates a new SummaryAgent.
func NewSummaryAgent() (*SummaryAgent, error) {
	systemPrompt, err := os.ReadFile("./system_prompts.txt")
	if err != nil {
		return nil, err
	}

	agent := blades.NewAgent("Summary Agent",
		blades.WithModel(os.Getenv("OPENAI_MODEL")), blades.WithProvider(bladesopenai.NewChatProvider()))

	return &SummaryAgent{
		agent:        agent,
		systemPrompt: string(systemPrompt),
	}, nil
}

// Summary summarizes the given content.
func (a *SummaryAgent) Summary(ctx context.Context, content string) (string, error) {
	prompt, err := blades.NewPromptTemplate().System(
		a.systemPrompt, nil).User("Please summarize the following blog post: {{.context}}", map[string]any{
		"context": content,
	}).Build()
	if err != nil {
		return "", err
	}

	result, err := a.agent.Run(ctx, prompt)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}

// SummaryResult is the result of a summary.
type SummaryResult struct {
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	PublishedAt time.Time `json:"published_at"`
}

func main() {
	summaryAgent, err := NewSummaryAgent()
	if err != nil {
		panic(err)
	}

	listCollector := colly.NewCollector()
	articleCollector := colly.NewCollector()

	results := make([]*SummaryResult, 0)

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
		results = append(results, &SummaryResult{
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

	if len(results) == 0 {
		return
	}

	monthStr := time.Now().UTC().Format("200601")
	_, err = os.Stat(path.Join("./data", monthStr))
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path.Join("./data", monthStr), 0755)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	dataStr := time.Now().UTC().Format("20060102")
	filename := path.Join("./data", monthStr, dataStr+".jsonl")
	var f *os.File
	_, err = os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Create(filename)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} else {
		f, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
	}
	defer f.Close()

	for _, result := range results {
		err = json.NewEncoder(f).Encode(result)
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(context.Background(), "save results",
		slog.String("Filename", filename),
		slog.Int("Rows", len(results)))
}
