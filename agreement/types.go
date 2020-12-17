package agreement

import "fmt"

const (
	StepReductionOne = "REDUCTION_ONE"
	StepReductionTwo = "REDUCTION_TWO"

	Step      = "STEP"
	StepFinal = "STEP_FINAL"

	RoleProposer  = "PROPOSER"
	RoleCommittee = "COMMITTEE"
)

// Vote defines agreement vote structure
type Vote struct {
	Issuer        []byte
	VrfHash       []byte
	VrfProof      []byte
	VoteCount     uint64
	Round         int
	Step          string
	LastBlockHash []byte
	SelectedBlock [][]byte
	Signature     []byte
}

func (v Vote) hashString() string {
	//SenderPK|VrfHash|VrfProof|VoteCount|Round|Step|LastBlockHash|SelectedBlock
	return fmt.Sprintf("%x|%x|%x|%d|%d|%s|%x|%x", v.Issuer, v.VrfHash, v.VrfProof, v.VoteCount, v.Round, v.Step, v.LastBlockHash, v.SelectedBlock)
}

// Hash calculatest he hash of the Vote
func (v Vote) Hash() []byte {
	hashString := v.hashString()
	return digest([]byte(hashString))
}

type context struct {
	publickKey []byte
	privateKey []byte
}

// Proposal defines agreement proposal structure
type Proposal struct {
	Issuer    []byte
	Index     int
	PrevHash  []byte
	SeedProof []byte
	VrfProof  []byte
	BlockHash []byte
	Signature []byte
}

func (p *Proposal) hashString() string {
	//SenderPK|VrfHash|VrfProof|VoteCount|Round|Step|LastBlockHash|SelectedBlock
	return fmt.Sprintf("%x|%d|%x|%x|%x|%x|%x", p.Issuer, p.Index, p.PrevHash, p.SeedProof, p.VrfProof, p.BlockHash)
}

// Hash calculates SHA 256 digest of the proposal
func (p *Proposal) Hash() []byte {
	hashString := p.hashString()
	return digest([]byte(hashString))
}
