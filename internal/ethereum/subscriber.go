package ethereum

import "context"

// BlockSubscriber streams new Ethereum block headers.
type BlockSubscriber interface {
	Subscribe(ctx context.Context) (<-chan Block, error)
}
