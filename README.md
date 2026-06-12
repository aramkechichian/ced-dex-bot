# CED-DEX Arbitrage Bot

Real-time arbitrage detection between Binance (CEX) and Uniswap V3 (DEX) for ETH-USDC.

## Phase 0 — Bootstrap

Project skeleton with configuration loading and Docker packaging.

## Prerequisites

- Go 1.21+
- Docker (optional, for containerized run)
- Infura or Alchemy API key

## Quick start (local)

```bash
cp .env.example .env
# Edit .env and set INFURA_API_KEY

export $(grep -v '^#' .env | xargs)
go run ./cmd/bot
```

## Quick start (Docker)

```bash
cp .env.example .env
# Edit .env and set INFURA_API_KEY

docker compose up --build
```

## Configuration

Default config file: `configs/config.yaml`

Override with:

```bash
go run ./cmd/bot -config path/to/config.yaml
```

Environment variables referenced in config (e.g. `${INFURA_API_KEY}`) are expanded at load time.
