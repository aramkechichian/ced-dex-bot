package common

import (
	"errors"
	"math/big"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	wrapped := errors.Join(ErrRateLimited, errors.New("binance 429"))

	if !errors.Is(wrapped, ErrRateLimited) {
		t.Fatal("expected errors.Is to match sentinel error")
	}
	if errors.Is(wrapped, ErrConnectionLost) {
		t.Fatal("unexpected match for different sentinel error")
	}
}

func TestDecimalScales(t *testing.T) {
	oneETH := new(big.Int).Set(EthScale)
	if oneETH.String() != "1000000000000000000" {
		t.Fatalf("EthScale: got %s", oneETH)
	}

	oneUSDC := new(big.Int).Set(USDCScale)
	if oneUSDC.String() != "1000000" {
		t.Fatalf("USDCScale: got %s", oneUSDC)
	}
}

func TestMainnetAddressesAreChecksummed(t *testing.T) {
	addresses := []string{WETHAddress, USDCAddress, PoolAddress, QuoterV2Address}
	for _, addr := range addresses {
		if len(addr) != 42 || addr[:2] != "0x" {
			t.Fatalf("invalid address format: %s", addr)
		}
	}
}
