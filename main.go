// Package main ...
package main

import (
	"context"

	airapp "github.com/anyvoxel/airmid/app"

	_ "github.com/anyvoxel/vela/pkg/app"
	_ "github.com/anyvoxel/vela/pkg/collectors/allthingsdistributed"
	_ "github.com/anyvoxel/vela/pkg/collectors/amazonscience"
	_ "github.com/anyvoxel/vela/pkg/collectors/bravenewgeek"
	_ "github.com/anyvoxel/vela/pkg/collectors/brendangregg"
	_ "github.com/anyvoxel/vela/pkg/collectors/brooker"
	_ "github.com/anyvoxel/vela/pkg/collectors/charap"
	_ "github.com/anyvoxel/vela/pkg/collectors/cloudflareblog"
	_ "github.com/anyvoxel/vela/pkg/collectors/emptysqua"
	_ "github.com/anyvoxel/vela/pkg/collectors/engineeringfb"
	_ "github.com/anyvoxel/vela/pkg/collectors/googleblog"
	_ "github.com/anyvoxel/vela/pkg/collectors/jackvanlightly"
	_ "github.com/anyvoxel/vela/pkg/collectors/micahlerner"
	_ "github.com/anyvoxel/vela/pkg/collectors/muratbuffalo"
	_ "github.com/anyvoxel/vela/pkg/collectors/mydistributed"
	_ "github.com/anyvoxel/vela/pkg/collectors/researchrsc"
	_ "github.com/anyvoxel/vela/pkg/collectors/shopifyblog"
	_ "github.com/anyvoxel/vela/pkg/collectors/thegreenplace"
	_ "github.com/anyvoxel/vela/pkg/collectors/uberblog"
)

func main() {
	err := airapp.Run(context.Background())
	if err != nil {
		panic(err)
	}
}
