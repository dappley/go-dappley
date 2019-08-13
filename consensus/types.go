package consensus

import "github.com/dappley/go-dappley/core/block"

type Process func(ctx *block.Block)
