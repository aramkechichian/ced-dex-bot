package common

import "math/big"

// Token decimals on Ethereum mainnet.
// ERC-20 amounts are integers; decimals tell you how to interpret them.
//   1 ETH  = 1 × 10^18 wei
//   1 USDC = 1 × 10^6  micro-USDC
const (
	ETHDecimals  = 18
	USDCDecimals = 6
)

// Uniswap V3 pool fee tier.
// 3000 = 0.3% (fee is expressed in hundredths of a bip).
const UniswapFeeBps uint32 = 3000

// Estimated gas units for a Uniswap V3 swap.
// Used later to convert gas cost to USD in the arbitrage detector.
// Real swaps vary; this is a reasonable mainnet estimate.
const EstimatedSwapGasUnits uint64 = 180_000

// DefaultBinanceTakerFee is the fallback taker fee (0.1%) when not set in config.
const DefaultBinanceTakerFee = 0.001

// Ethereum mainnet contract addresses.
// These are fixed on-chain identifiers — they do not change at runtime.
const (
	WETHAddress     = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	USDCAddress     = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	PoolAddress     = "0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640" // ETH-USDC 0.3%
	QuoterV2Address = "0xb27308f9F90D607463bb33eA1BeBb41C27CE5AB6"
)

// Scales for converting raw on-chain integers to human-readable amounts.
// Pre-computed once to avoid repeated exponentiation in hot paths.
var (
	EthScale  = new(big.Int).Exp(big.NewInt(10), big.NewInt(ETHDecimals), nil)
	USDCScale = new(big.Int).Exp(big.NewInt(10), big.NewInt(USDCDecimals), nil)
)
