package ethereum

import "time"

// Block represents an Ethereum block header used as the pipeline trigger.
// Every arbitrage evaluation is anchored to a specific block number so
// CEX and DEX snapshots stay consistent.
type Block struct {
	Number    uint64
	Hash      string
	Timestamp time.Time
}
