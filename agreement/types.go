package agreement

const (
	StepReductionOne = "REDUCTION_ONE"
	StepReductionTwo = "REDUCTION_TWO"

	Step      = "STEP"
	StepFinal = "STEP_FINAL"

	RoleProposer  = "PROPOSER"
	RoleCommittee = "COMMITTEE"
)

type Vote struct {
	SenderPK      []byte
	VrfHash       []byte
	VrfProof      []byte
	VoteCount     int
	Round         int
	Step          string
	LastBlockHash []byte
	SelectedBlock []byte
}

type ProtocolParams struct {
	UserMoney  int
	TotalMoney int

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
