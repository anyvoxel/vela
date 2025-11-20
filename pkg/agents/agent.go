// Package agents contains implementation of llm summarizer.
package agents

import (
	"context"
	_ "embed"
	"os"
	"reflect"

	"github.com/anyvoxel/airmid/anvil"
	airapp "github.com/anyvoxel/airmid/app"
	"github.com/anyvoxel/airmid/ioc"

	"github.com/go-kratos/blades"
	bladesopenai "github.com/go-kratos/blades/contrib/openai"
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
	agent        *blades.Agent
	systemPrompt string
}

var (
	_ ioc.InitializingBean = (*Summarizer)(nil)
)

// AfterPropertiesSet implement InitializingBean
func (a *Summarizer) AfterPropertiesSet(context.Context) error {
	agent := blades.NewAgent("Summary Agent",
		blades.WithModel(os.Getenv("OPENAI_MODEL")), blades.WithProvider(bladesopenai.NewChatProvider()))
	a.agent = agent
	a.systemPrompt = systemPrompts
	return nil
}

// Summary summarizes the given content.
func (a *Summarizer) Summary(ctx context.Context, content string) (string, error) {
	prompt, err := blades.NewPromptTemplate().System(
		a.systemPrompt, nil).User("Please summarize the following blog post: {{.content}}", map[string]any{
		"content": content,
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
