package uniswap

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// revertReturnData extracts ABI-encoded return data from a QuoterV2 eth_call revert.
// QuoterV2 intentionally reverts to return quote results (off-chain simulation pattern).
func revertReturnData(err error) ([]byte, bool) {
	if err == nil {
		return nil, false
	}

	var dataErr rpc.DataError
	if errors.As(err, &dataErr) {
		return decodeErrorData(dataErr.ErrorData())
	}

	// Some providers embed hex revert data in the error string.
	msg := err.Error()
	if idx := strings.LastIndex(msg, "0x"); idx != -1 {
		hex := msg[idx:]
		if end := strings.IndexAny(hex, " "); end != -1 {
			hex = hex[:end]
		}
		if len(hex) > 2 {
			return common.FromHex(hex), true
		}
	}

	return nil, false
}

func decodeErrorData(data interface{}) ([]byte, bool) {
	switch v := data.(type) {
	case string:
		if strings.HasPrefix(v, "0x") && len(v) > 2 {
			return common.FromHex(v), true
		}
	case map[string]interface{}:
		if raw, ok := v["data"].(string); ok {
			return decodeErrorData(raw)
		}
	case json.RawMessage:
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			return decodeErrorData(s)
		}
	}
	return nil, false
}
