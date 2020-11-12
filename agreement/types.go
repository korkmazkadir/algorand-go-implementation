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

type Vote struct {
	Issuer        []byte
	VrfHash       []byte
	VrfProof      []byte
	VoteCount     uint64
	Round         int
	Step          string
	LastBlockHash []byte
	SelectedBlock []byte
	//------------------//
	Signature []byte
}

func (v Vote) hashString() string {
	//SenderPK|VrfHash|VrfProof|VoteCount|Round|Step|LastBlockHash|SelectedBlock
	return fmt.Sprintf("%x|%x|%x|%d|%d|%s|%x|%x", v.Issuer, v.VrfHash, v.VrfProof, v.VoteCount, v.Round, v.Step, v.LastBlockHash, v.SelectedBlock)
}

func (v Vote) Hash() []byte {
	hashString := v.hashString()
	return digest([]byte(hashString))
}

type ProtocolParams struct {
	UserMoney  uint64
	TotalMoney uint64

	ThresholdProposer int
	TSmallStep        int
	TBigStep          float32
	TBigFinal         int
	TSmallFinal       float32

	BlockSizeInBytes int

	LamdaPriority int
	LamdaBlock    int
	LamdaStep     int
	LamdaStepVar  int
}

type context struct {
	publickKey []byte
	privateKey []byte
}
