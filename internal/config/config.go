package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Ethereum    EthereumConfig    `yaml:"ethereum"`
	Binance     BinanceConfig     `yaml:"binance"`
	Uniswap     UniswapConfig     `yaml:"uniswap"`
	Arbitrage   ArbitrageConfig   `yaml:"arbitrage"`
	Resilience  ResilienceConfig  `yaml:"resilience"`
	Logging     LoggingConfig     `yaml:"logging"`
}

type EthereumConfig struct {
	WSURL             string `yaml:"ws_url"`
	HTTPURL           string `yaml:"http_url"`
	MaxBackfillBlocks uint64 `yaml:"max_backfill_blocks"`
}

func (c EthereumConfig) MaxBackfillBlocksOrDefault() uint64 {
	if c.MaxBackfillBlocks == 0 {
		return 10
	}
	return c.MaxBackfillBlocks
}

type BinanceConfig struct {
	BaseURL        string `yaml:"base_url"`
	Symbol         string `yaml:"symbol"`
	OrderbookLimit int    `yaml:"orderbook_limit"`
}

type UniswapConfig struct {
	QuoterAddress string `yaml:"quoter_address"`
	PoolAddress   string `yaml:"pool_address"`
	WETHAddress   string `yaml:"weth_address"`
	USDCAddress   string `yaml:"usdc_address"`
	Fee           uint32 `yaml:"fee"`
}

type ArbitrageConfig struct {
	TradeSizesETH   []float64 `yaml:"trade_sizes_eth"`
	MinProfitPct    float64   `yaml:"min_profit_pct"`
	BinanceTakerFee float64   `yaml:"binance_taker_fee"`
}

type ResilienceConfig struct {
	GasCacheTTLSeconds int     `yaml:"gas_cache_ttl_seconds"`
	BinanceRPS         float64 `yaml:"binance_rps"`
	BinanceBurst       int     `yaml:"binance_burst"`
	RPCMaxRetries      int     `yaml:"rpc_max_retries"`
	RPCRetryBaseMS     int     `yaml:"rpc_retry_base_ms"`
}

func (r ResilienceConfig) GasCacheTTLSecondsOrDefault() int {
	if r.GasCacheTTLSeconds <= 0 {
		return 12
	}
	return r.GasCacheTTLSeconds
}

func (r ResilienceConfig) BinanceRPSOrDefault() float64 {
	if r.BinanceRPS <= 0 {
		return 5
	}
	return r.BinanceRPS
}

func (r ResilienceConfig) BinanceBurstOrDefault() int {
	if r.BinanceBurst <= 0 {
		return 10
	}
	return r.BinanceBurst
}

func (r ResilienceConfig) RPCMaxRetriesOrDefault() int {
	if r.RPCMaxRetries <= 0 {
		return 3
	}
	return r.RPCMaxRetries
}

func (r ResilienceConfig) RPCRetryBaseMSOrDefault() int {
	if r.RPCRetryBaseMS <= 0 {
		return 200
	}
	return r.RPCRetryBaseMS
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	var missing []string

	if strings.TrimSpace(c.Binance.Symbol) == "" {
		missing = append(missing, "binance.symbol")
	}
	if len(c.Arbitrage.TradeSizesETH) == 0 {
		missing = append(missing, "arbitrage.trade_sizes_eth")
	}

	if len(missing) > 0 {
		return fmt.Errorf("invalid config: missing or empty fields: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ValidateLive checks Ethereum RPC settings required for mainnet mode.
func (c *Config) ValidateLive() error {
	var missing []string

	if strings.TrimSpace(c.Ethereum.WSURL) == "" || hasMissingAPIKey(c.Ethereum.WSURL) {
		missing = append(missing, "ethereum.ws_url (set INFURA_API_KEY)")
	}
	if strings.TrimSpace(c.Ethereum.HTTPURL) == "" || hasMissingAPIKey(c.Ethereum.HTTPURL) {
		missing = append(missing, "ethereum.http_url (set INFURA_API_KEY)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("invalid live config: missing or empty fields: %s", strings.Join(missing, ", "))
	}

	return nil
}

func hasMissingAPIKey(url string) bool {
	return strings.Contains(url, "${") || strings.HasSuffix(url, "/v3/")
}
