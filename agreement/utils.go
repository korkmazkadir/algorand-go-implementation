package agreement

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"

	"../blockchain"
)

func digest(input []byte) []byte {
	h := sha256.New()
	h.Write(input)
	return h.Sum(nil)
}

// ByteToBase64String converts a byte slice to a base64 string
func ByteToBase64String(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func compareBlocks(a *blockchain.Block, b *blockchain.Block) int {

	hashA := a.Hash()
	hashB := b.Hash()

	return bytes.Compare(hashA, hashB)
}

func signBlock(block *blockchain.Block, privateKey []byte) {
	blockHash := block.Hash()
	block.Signature = ed25519.Sign(privateKey, blockHash)
}

func signVote(vote *Vote, privateKey []byte) {
	voteHash := vote.Hash()
	vote.Signature = ed25519.Sign(privateKey, voteHash)
}

func isBlockSignatureValid(block *blockchain.Block) bool {
	blockHash := block.Hash()
	return ed25519.Verify(block.Issuer, blockHash, block.Signature)
}

func isVoteSignatureValid(vote *Vote) bool {
	voteHash := vote.Hash()
	return ed25519.Verify(vote.Issuer, voteHash, vote.Signature)
}
