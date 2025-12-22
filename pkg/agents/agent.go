// Package agents contains implementation of llm summarizer.
package agents

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"reflect"

	"github.com/anyvoxel/airmid/anvil"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"

	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/contrib/openai"
)

func init() {
	anvil.Must(airapp.RegisterBeanDefinition(
		"vela.agents.summarizer",
		ioc.MustNewBeanDefinition(
			reflect.TypeOf((*Summarizer)(nil)),
		),
	))
}

//go:embed system_prompts.md
var systemPrompts string

// Summarizer is a agent that can summarize a blog post.
type Summarizer struct {
	agent        blades.Agent
	systemPrompt string
}

var (
	_ ioc.InitializingBean = (*Summarizer)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (a *Summarizer) AfterPropertiesSet(context.Context) error {
	model := openai.NewModel(os.Getenv("OPENAI_MODEL"), openai.Config{
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	agent, err := blades.NewAgent("Summary Agent",
		blades.WithModel(model), blades.WithInstruction(systemPrompts))
	if err != nil {
		return err
	}

	a.agent = agent
	a.systemPrompt = systemPrompts
	return nil
}

// Summary summarizes the given content.
func (a *Summarizer) Summary(ctx context.Context, content string) (string, error) {
	prompt := blades.UserMessage(fmt.Sprintf("Please summarize the following blog post: %s", content))
	runner := blades.NewRunner(a.agent)
	result, err := runner.Run(ctx, prompt)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}
