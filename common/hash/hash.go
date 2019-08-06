package hash

import (
	"bytes"
	"encoding/hex"
)

type Hash []byte

func (h Hash) String() string {
	return hex.EncodeToString(h)
}

func (h Hash) Equals(nh Hash) bool {
	return bytes.Compare(h, nh) == 0
}
