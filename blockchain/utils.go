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

// XORBytes takes two byte slices and XORs them together, returning the final
// byte slice. It is an error to pass in two byte slices that do not have the
// same length.
// https://github.com/hashicorp/vault/blob/v1.6.1/helper/xor/xor.go#L13
func XORBytes(a, b []byte) ([]byte, error) {
	if len(a) != len(b) {
		return nil, fmt.Errorf("length of byte slices is not equivalent: %d != %d", len(a), len(b))
	}

	buf := make([]byte, len(a))

	for i := range a {
		buf[i] = a[i] ^ b[i]
	}

	return buf, nil
}
