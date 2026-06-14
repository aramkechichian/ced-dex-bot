package uniswap

import (
	"errors"
	"testing"
)

type mockDataError struct {
	msg  string
	data interface{}
}

func (e mockDataError) Error() string  { return e.msg }
func (e mockDataError) ErrorData() interface{} { return e.data }

func TestRevertReturnDataFromRPCError(t *testing.T) {
	err := mockDataError{
		msg:  "execution reverted",
		data: "0x000000000000000000000000000000000000000000000086a94d4f000000",
	}

	data, ok := revertReturnData(err)
	if !ok {
		t.Fatal("expected revert data")
	}
	if len(data) == 0 {
		t.Fatal("empty revert data")
	}
}

func TestRevertReturnDataNilError(t *testing.T) {
	_, ok := revertReturnData(nil)
	if ok {
		t.Fatal("expected false for nil error")
	}
}

func TestRevertReturnDataFromErrorString(t *testing.T) {
	err := errors.New(`execution reverted: 0x000000000000000000000000000000000000000000000086a94d4f000000`)
	data, ok := revertReturnData(err)
	if !ok || len(data) == 0 {
		t.Fatalf("expected data from string, ok=%v len=%d", ok, len(data))
	}
}
