package arbitrage

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestDirectionString(t *testing.T) {
	if CEXToDEX.String() != "CEX → DEX (Buy on Binance, Sell on Uniswap)" {
		t.Fatalf("unexpected CEXToDEX: %s", CEXToDEX.String())
	}
	if DEXToCEX.String() != "DEX → CEX (Buy on Uniswap, Sell on Binance)" {
		t.Fatalf("unexpected DEXToCEX: %s", DEXToCEX.String())
	}
}

func TestOpportunityFields(t *testing.T) {
	opp := Opportunity{
		BlockNumber:  18234567,
		Direction:    CEXToDEX,
		TradeSizeETH: decimal.NewFromInt(10),
		CEXPriceUSD:  decimal.NewFromFloat(2245.30),
		DEXPriceUSD:  decimal.NewFromFloat(2267.80),
		ProfitUSD:    decimal.NewFromFloat(225.00),
	}

	if opp.Direction != CEXToDEX {
		t.Fatal("direction mismatch")
	}
	if opp.TradeSizeETH.String() != "10" {
		t.Fatalf("trade size: %s", opp.TradeSizeETH)
	}
}
