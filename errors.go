package decimal

import (
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

// RevertError exposes only decoded ABI data, never raw RPC URLs or credentials.
type RevertError struct {
	Name      string
	Arguments []any
	Reason    string
}

func (e *RevertError) Error() string { return "contract reverted: " + e.Name }
func decodeRPCError(err error, a abi.ABI) error {
	var data interface{ ErrorData() any }
	if errors.As(err, &data) {
		s, ok := data.ErrorData().(string)
		if ok {
			b, e := hex.DecodeString(strings.TrimPrefix(s, "0x"))
			if e == nil && len(b) >= 4 {
				if reason, e := abi.UnpackRevert(b); e == nil {
					return &RevertError{Name: "Error/Panic", Reason: reason}
				}
				for _, custom := range a.Errors {
					if string(custom.ID[:4]) == string(b[:4]) {
						v, e := custom.Inputs.Unpack(b[4:])
						if e == nil {
							return &RevertError{Name: custom.Name, Arguments: v}
						}
					}
				}
			}
		}
	}
	return errors.New("contract RPC call failed")
}
