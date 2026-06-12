package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Ethereum  EthereumConfig  `yaml:"ethereum"`
	Binance   BinanceConfig   `yaml:"binance"`
	Uniswap   UniswapConfig   `yaml:"uniswap"`
	Arbitrage ArbitrageConfig `yaml:"arbitrage"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type EthereumConfig struct {
	WSURL   string `yaml:"ws_url"`
	HTTPURL string `yaml:"http_url"`
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

	if strings.TrimSpace(c.Ethereum.WSURL) == "" || hasMissingAPIKey(c.Ethereum.WSURL) {
		missing = append(missing, "ethereum.ws_url (set INFURA_API_KEY)")
	}
	if strings.TrimSpace(c.Ethereum.HTTPURL) == "" || hasMissingAPIKey(c.Ethereum.HTTPURL) {
		missing = append(missing, "ethereum.http_url (set INFURA_API_KEY)")
	}
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

func hasMissingAPIKey(url string) bool {
	return strings.Contains(url, "${") || strings.HasSuffix(url, "/v3/")
}
