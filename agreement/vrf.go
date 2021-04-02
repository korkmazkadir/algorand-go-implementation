package agreement

import (
	"crypto/ed25519"
)

type vrf struct {
	privateKey []byte
}

func (v vrf) ProduceProof(input []byte) ([]byte, []byte) {
	proof := ed25519.Sign(v.privateKey, digest(input))
	hash := digest(proof)
	return hash, proof
}

func (v vrf) Verify(publicKey []byte, input []byte, proof []byte) bool {

	if ed25519.Verify(publicKey, digest(input), proof) == false {
		return false
	}

	return true
}
