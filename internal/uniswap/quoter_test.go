package uniswap

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
)

func TestEffectiveSellPrice(t *testing.T) {
	oneETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	usdcOut := big.NewInt(2_671_340_000) // 2671.34 USDC

	price := effectiveSellPrice(oneETH, usdcOut)
	want := decimal.RequireFromString("2671.34")
	if !price.Equal(want) {
		t.Fatalf("got %s want %s", price, want)
	}
}

func TestEffectiveBuyPrice(t *testing.T) {
	oneETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	usdcIn := big.NewInt(2_672_000_000) // 2672.00 USDC

	price := effectiveBuyPrice(usdcIn, oneETH)
	want := decimal.RequireFromString("2672.00")
	if !price.Equal(want) {
		t.Fatalf("got %s want %s", price, want)
	}
}

func TestETHToWei(t *testing.T) {
	wei := ETHToWei(decimal.NewFromInt(1))
	if wei.Cmp(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) != 0 {
		t.Fatalf("unexpected wei: %s", wei)
	}
}

func TestRawToDecimal(t *testing.T) {
	v := rawToDecimal(big.NewInt(1_500_000), 6)
	if !v.Equal(decimal.NewFromFloat(1.5)) {
		t.Fatalf("got %s", v)
	}
}
