package agreement

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"../blockchain"
)

func digest(input []byte) []byte {
	h := sha256.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}

// ByteToBase64String converts a byte slice to a base64 string
func ByteToBase64String(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func compareBlocks(a *blockchain.Block, b *blockchain.Block) int {
	hashA := a.Hash()
	hashB := a.Hash()
	return bytes.Compare(hashA, hashB)
}

func emptyHash(round int, previousBlockHash []byte) []byte {
	input := fmt.Sprintf("%d|%x", round, previousBlockHash)
	return digest([]byte(input))
}
