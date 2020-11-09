package blockchain

import (
	"crypto/sha256"
	"fmt"
)

func hash(o interface{}) []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", o)))
	return h.Sum(nil)
}

func digest(input []byte) []byte {
	h := sha256.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}
