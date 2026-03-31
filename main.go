// Package main ...
package main

import (
	"context"

	airapp "github.com/anyvoxel/airmid/app"

	_ "github.com/anyvoxel/vela/pkg/app"
)

func main() {
	err := airapp.Run(context.Background())
	if err != nil {
		panic(err)
	}
}
