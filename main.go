// Package main ...
package main

import (
	"context"

	"github.com/anyvoxel/vela/pkg/app"
)

func main() { //nolint
	a, err := app.NewApplication(context.Background())
	if err != nil {
		panic(err)
	}

	err = a.Start(context.Background())
	if err != nil {
		panic(err)
	}
}
