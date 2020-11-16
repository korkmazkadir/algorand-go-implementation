package agreement

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"../blockchain"
	"../config"
	"../filter"
	"github.com/korkmazkadir/go-rpc-node/node"
)

// TODO: tag field of message can be a byte
const tagBlock = "B"
const tagVote = "V"
const tagProposal = "P"

// BAStar implements algorand agreement protocol
type BAStar struct {
	id int
	applicationImp

	memoryPool blockchain.MemoryPool
	blockchain blockchain.Blockchain

	localVote  *Vote
	emptyBlock *blockchain.Block

	waitingBlockMap map[string]*blockchain.Block

	sortition *sortition
	params    config.ProtocolParams
	context
	wg  *sync.WaitGroup
	log *log.Logger
}

// NewBAStar creates an instance of agrreement protocol
func NewBAStar(params config.ProtocolParams, publicKey []byte, privateKey []byte, memoryPool blockchain.MemoryPool, blockchain blockchain.Blockchain, logger *log.Logger) *BAStar {
	ba := new(BAStar)
	ba.networkReadySig = make(chan struct{}, 1)

	// TTL 60 seconds
	ba.messageFilter = filter.NewUniqueMessageFilter(60)

	ba.demultiplexer = newDemux(1)
	ba.outgoingMessages = make(chan node.Message, 100)

	ba.params = params
	ba.publickKey = publicKey
	ba.privateKey = privateKey
	ba.memoryPool = memoryPool
	ba.blockchain = blockchain
	ba.sortition = newSortition(ba.privateKey)

	ba.wg = &sync.WaitGroup{}

	ba.log = logger

	return ba
}

// Start runs the agreement protocol
func (ba *BAStar) Start() {

	go func() {
		//waits for network signal
		<-ba.networkReadySig

		time.Sleep(30 * time.Second)

		ba.log.Println("Started...")

		ba.wg.Add(1)
		ba.mainLoop()
	}()

}

func (ba *BAStar) mainLoop() {

	startTime := time.Now()

	for {

		currentRound := (ba.blockchain.GetBlockHeight() - 1)
		ba.log.Printf("=====================> Round %d Finished in %f seconds <=====================\n", currentRound, time.Since(startTime).Seconds())
		startTime = time.Now()

		//creates an empty block for the current round to use
		ba.createEmptyBlock()

		ba.log.Printf("Last Block Hash %s \n", ByteToBase64String(ba.blockchain.GetLastBlockHash()))

		round := ba.blockchain.GetBlockHeight()

		proposedBlock := ba.proposeBlock()

		localProposal := ba.submitProposal(proposedBlock)
		highestPriorityProposal := ba.waitForProposals(localProposal)

		// If highestPriorityProposal is the proposal for the current block
		if proposedBlock != nil && highestPriorityProposal != nil && bytes.Equal(proposedBlock.Hash(), highestPriorityProposal.BlockHash) {
			start := time.Now()
			ba.BroadcastBlock(*proposedBlock)
			ba.log.Printf("Block broadcasted: %s Elapsed time %f \n", ByteToBase64String(highestPriorityProposal.BlockHash), time.Since(start).Seconds())
		}

		var blockHash []byte

		if highestPriorityProposal == nil {
			blockHash = ba.emptyBlock.Hash()
		} else {
			blockHash = highestPriorityProposal.BlockHash
		}

		blockHash = ba.reduction(round, blockHash)
		ba.log.Printf("Result of reduction: %s\n", ByteToBase64String(blockHash))

		blockHash = ba.binaryBA(round, blockHash)
		ba.log.Printf("Result of binary BA: %s\n", ByteToBase64String(blockHash))

		r, _ := ba.countVotes(round, StepFinal, ba.params.TSmallFinal, ba.params.TBigFinal, ba.params.LamdaStep)

		if bytes.Equal(r, blockHash) {

			ba.log.Printf("FINAL CONSENSUS on %s\n", ByteToBase64String(blockHash))

			if proposedBlock != nil && highestPriorityProposal != nil && bytes.Equal(proposedBlock.Hash(), highestPriorityProposal.BlockHash) {
				err := ba.blockchain.AppendBlock(*proposedBlock)
				if err != nil {
					panic(err)
				}
			} else if bytes.Equal(ba.emptyBlock.Hash(), blockHash) {
				ba.log.Println("Appending empty block to the blockchain!!")
				err := ba.blockchain.AppendBlock(*ba.emptyBlock)
				if err != nil {
					panic(err)
				}
			} else {
				highestPriorityBlock := ba.waitForMissingBlock(round, blockHash)
				err := ba.blockchain.AppendBlock(*highestPriorityBlock)
				if err != nil {
					panic(err)
				}
			}

		} else {

			ba.log.Printf("TENTATIVE CONSENSUS on %s\n", ByteToBase64String(blockHash))

			var missingBlock *blockchain.Block
			if bytes.Equal(ba.emptyBlock.Hash(), blockHash) {
				ba.log.Println("Appending empty block to the blockchain!!")
				missingBlock = ba.emptyBlock
			} else {
				missingBlock = ba.waitForMissingBlock(round, blockHash)
			}

			err := ba.blockchain.AppendBlock(*missingBlock)
			if err != nil {
				panic(err)
			}

		}

		//sets round on demux
		ba.demultiplexer.SetRound(ba.blockchain.GetBlockHeight())

	}
}

func (ba *BAStar) waitForMissingBlock(round int, blockHash []byte) *blockchain.Block {

	start := time.Now()

	ba.log.Printf("Waiting for missing block %s\n", ByteToBase64String(blockHash))

	if bytes.Equal(blockHash, ba.emptyBlock.Hash()) {
		//panic("Waiting for empty block!!!")
		ba.log.Println("WARNING: returning empty block to append the chain. Be careful!!!")
		return ba.emptyBlock
	}

	blockChan := ba.demultiplexer.GetBlockChan(round)

	for {

		select {
		case incommingBlock := <-blockChan:

			block := incommingBlock.block
			forwardBlock := incommingBlock.forward

			if bytes.Equal(block.Hash(), blockHash) {
				forwardBlock()
				ba.log.Printf("Missing block received %s Time elpased: %f \n", ByteToBase64String(blockHash), time.Since(start).Seconds())
				return &block
			}

			ba.log.Printf("Discarting block %s round: %d \n", ByteToBase64String(block.Hash()), block.Index)

		}
	}

}

func (ba *BAStar) waitForProposals(localProposal *Proposal) *Proposal {

	ba.log.Printf("Waiting for proposals...")

	sleepTime := time.Duration(ba.params.LamdaPriority + ba.params.LamdaStepVar)
	timeout := time.After(sleepTime * time.Second)

	proposalChan := ba.demultiplexer.GetProposalChan(ba.blockchain.GetBlockHeight())

	var highestPriorityProposal = localProposal

	for {
		select {
		case incommingProposal := <-proposalChan:

			proposal := incommingProposal.proposal
			forwardProposal := incommingProposal.forward

			if highestPriorityProposal == nil || (compareProposals(highestPriorityProposal, &proposal) < 0) {
				ba.log.Printf("Forwarding proposal for the block: %s\n", ByteToBase64String(proposal.BlockHash))
				highestPriorityProposal = &proposal
				forwardProposal()
			} else {
				ba.log.Printf("Proposal Not forwarded for the block %s\n", ByteToBase64String(proposal.BlockHash))
			}

		case <-timeout:

			if highestPriorityProposal == nil {
				ba.log.Println("No proposal received!!!")
			} else {
				ba.log.Printf("Highest priority proposal for the block %s \n", ByteToBase64String(highestPriorityProposal.BlockHash))
			}

			return highestPriorityProposal
		}
	}

}

func (ba *BAStar) waitForBlocks(proposedBlock *blockchain.Block) *blockchain.Block {

	ba.log.Printf("Waiting for proposals...")

	sleepTime := time.Duration(ba.params.LamdaPriority + ba.params.LamdaStepVar)
	timeout := time.After(sleepTime * time.Second)

	blockChan := ba.demultiplexer.GetBlockChan(ba.blockchain.GetBlockHeight())

	ba.waitingBlockMap = make(map[string]*blockchain.Block)

	var highestPriorityBlock = proposedBlock

	for {

		select {
		case incommingBlock := <-blockChan:

			block := incommingBlock.block
			forwardBlock := incommingBlock.forward

			//TODO write a valdate method for blocks

			// puts received block to waiting block list
			ba.waitingBlockMap[string(block.Hash())] = &block

			if highestPriorityBlock == nil || (compareBlocks(highestPriorityBlock, &block) < 0) {
				ba.log.Printf("Block forwarded: %s\n", ByteToBase64String(block.Hash()))
				highestPriorityBlock = &block
				forwardBlock()
			} else {
				ba.log.Printf("Not forwarded: %s result: %d \n", ByteToBase64String(block.Hash()), compareBlocks(highestPriorityBlock, &block))
			}

		case <-timeout:

			if highestPriorityBlock == nil {
				highestPriorityBlock = ba.emptyBlock
			}

			ba.log.Printf("Highest priority block: %s \n", ByteToBase64String(highestPriorityBlock.Hash()))

			return highestPriorityBlock
		}

	}

}

func (ba *BAStar) reduction(round int, blockHash []byte) []byte {

	stepThreshold := ba.params.TSmallStep
	ba.committeeVote(round, StepReductionOne, stepThreshold, blockHash)

	voteThreshold := ba.params.TBigStep
	timerValueOne := ba.params.LamdaBlock + ba.params.LamdaStep

	blockHash1, isTimerExpired := ba.countVotes(round, StepReductionOne, voteThreshold, stepThreshold, timerValueOne)
	emptyHash := ba.emptyBlock.Hash()

	if isTimerExpired {
		ba.committeeVote(round, StepReductionTwo, stepThreshold, emptyHash)
	} else {
		ba.committeeVote(round, StepReductionTwo, stepThreshold, blockHash1)
	}

	blockHash2, isTimerExpired := ba.countVotes(round, StepReductionTwo, voteThreshold, stepThreshold, ba.params.LamdaStep)

	if isTimerExpired {
		return emptyHash
	}

	return blockHash2
}

func (ba *BAStar) committeeVote(round int, step string, stepThreshold int, blockHash []byte) {

	lastBlock := ba.blockchain.GetLastBlock()
	seed := lastBlock.SeedHash

	role := fmt.Sprintf("%s|%d|%s", RoleCommittee, round, step)
	userMoney := ba.params.UserMoney
	totalMoney := ba.params.TotalMoney

	hash, proof, numberOfTimesSelected := ba.sortition.Select(string(seed), stepThreshold, role, userMoney, totalMoney)

	if numberOfTimesSelected == 0 {
		ba.localVote = nil
		return
	}

	vote := Vote{
		Issuer:        ba.publickKey,
		VrfHash:       hash,
		VrfProof:      proof,
		VoteCount:     numberOfTimesSelected,
		Round:         round,
		Step:          step,
		LastBlockHash: ba.blockchain.GetLastBlockHash(),
		SelectedBlock: blockHash,
	}

	//signs vote
	signVote(&vote, ba.privateKey)

	ba.log.Printf("Voting for step: %s block: %s \n", step, ByteToBase64String(vote.SelectedBlock))
	ba.BroadcastVote(vote)

	ba.localVote = &vote
}

func (ba *BAStar) countVotes(round int, step string, voteThreshold float32, stepThreshold int, timeout int) (blockHash []byte, isTimerExprired bool) {

	counts := make(map[string]uint64)
	voters := make(map[string][]byte)

	sleepTime := time.Duration(timeout)
	timer := time.After(sleepTime * time.Second)
	incommingVotes := ba.demultiplexer.GetVoteChan(round, step)

	if ba.localVote != nil {
		voters[string(ba.localVote.Issuer)] = ba.localVote.Issuer
		counts[string(ba.localVote.SelectedBlock)] = ba.localVote.VoteCount
		ba.log.Printf("Local Vote for %s --> count:%d\n", ByteToBase64String(ba.localVote.SelectedBlock), ba.localVote.VoteCount)
	}

	for {
		select {
		case incommingVote := <-incommingVotes:

			vote := incommingVote.vote
			forwardCallback := incommingVote.forward

			numVotes, selectedBlockHash, _ := ba.validateVote(vote, step, stepThreshold)

			if numVotes == 0 {
				continue
			}

			if voters[string(vote.Issuer)] != nil || numVotes == 0 {
				continue
			}

			voters[string(vote.Issuer)] = vote.Issuer
			counts[string(selectedBlockHash)] = counts[string(selectedBlockHash)] + numVotes

			//ba.log.Printf("Vote for %s --> count:%d\n", ByteToBase64String(selectedBlockHash), numVotes)

			forwardCallback()
			//ba.log.Println("Vote forwarded!")

			//TODO: Check this line, ceil float64 does not seems good
			if float64(counts[string(selectedBlockHash)]) > math.Ceil(float64(voteThreshold)*float64(stepThreshold)) {
				ba.log.Printf("A block has reached the target vote count: %s\n", ByteToBase64String(selectedBlockHash))
				ba.printVoteCount(counts)
				return selectedBlockHash, false
			}

		case <-timer:
			ba.log.Println("Timer expired for count votes.")
			ba.printVoteCount(counts)
			return nil, true
		}
	}

}

func (ba *BAStar) printVoteCount(voteCountMap map[string]uint64) {

	for blockHash, count := range voteCountMap {
		ba.log.Printf("[vote-count] Block: %s --> Votes: %d \n", ByteToBase64String([]byte(blockHash)), count)
	}

}

func (ba *BAStar) binaryBA(round int, blockHash []byte) []byte {

	step := 1
	r := blockHash
	emptyBlockHash := ba.emptyBlock.Hash()
	// TODO define a maxstep
	for step < 255 {

		ba.log.Printf("=======> Binary BA STEP %d \n", step)

		ba.committeeVote(round, strconv.Itoa(step), ba.params.TSmallStep, r)
		r, timerExpired := ba.countVotes(round, strconv.Itoa(step), ba.params.TBigStep, ba.params.TSmallStep, ba.params.LamdaStep)

		if timerExpired {
			r = blockHash
		} else if bytes.Equal(emptyBlockHash, r) == false {

			// TODO: I have removed this. Consider to open later
			//votes for the same block for next 3 rounds
			//for i := 1; i < 4; i++ {
			//	ba.committeeVote(round, strconv.Itoa(step+1), ba.params.TSmallStep, r)
			//}

			if step == 1 {
				ba.committeeVote(round, StepFinal, ba.params.TBigFinal, r)
			}

			return r
		}

		/***************************************/
		step++
		ba.log.Printf("=======> Binary BA STEP %d \n", step)
		ba.committeeVote(round, strconv.Itoa(step), ba.params.TSmallStep, r)
		r, timerExpired = ba.countVotes(round, strconv.Itoa(step), ba.params.TBigStep, ba.params.TSmallStep, ba.params.LamdaStep)
		if timerExpired {
			r = emptyBlockHash
		} else if bytes.Equal(r, emptyBlockHash) {

			// TODO: I have removed this. Consider to open later
			//for i := 1; i < 4; i++ {
			//	ba.committeeVote(round, strconv.Itoa(step+1), ba.params.TSmallStep, r)
			//}

			return r
		}

		/***************************************/

		step++
		ba.log.Printf("=======> Binary BA STEP %d \n", step)
		ba.committeeVote(round, strconv.Itoa(step), ba.params.TSmallStep, r)
		r, timerExpired = ba.countVotes(round, strconv.Itoa(step), ba.params.TBigStep, ba.params.TSmallStep, ba.params.LamdaStep)
		if timerExpired {
			//Common coin
			ba.log.Println("COMMON-COIN")
		}
		step++
	}

	return nil
}

/************************************************************************************************************************/

func (ba *BAStar) proposeBlock() *blockchain.Block {
	// TODO: seed and role can be byte slice
	seed := ba.blockchain.GetLastBlockSeedHash()
	threshold := ba.params.ThresholdProposer
	role := RoleProposer
	userMoney := ba.params.UserMoney
	totalMoney := ba.params.TotalMoney

	// TODO: make sure that casting seed to string does not create issue
	vrfHash, vrfProof, numberOfTimesSelected := ba.sortition.Select(string(seed), threshold, role, userMoney, totalMoney)
	if numberOfTimesSelected == 0 {
		return nil
	}

	previousBlockHash := ba.blockchain.GetLastBlockHash()
	blockIndex := ba.blockchain.GetBlockHeight()

	block := ba.memoryPool.CreateBlock(previousBlockHash, blockIndex)
	block.Issuer = ba.publickKey
	block.VrfHash = vrfHash
	block.VrfProof = vrfProof

	//calculates the seed for the block
	ba.calculateSeed(block, ba.blockchain.GetLastBlock())

	//signs block
	signBlock(block, ba.privateKey)

	//broadcasts the block
	//TODO: decide between using pointer vs value
	//ba.BroadcastBlock(*block)

	ba.log.Printf("Proposed block: %s (Not Broadcasted!!!)\n", ByteToBase64String(block.Hash()))

	return block
}

func (ba *BAStar) submitProposal(proposedBlock *blockchain.Block) *Proposal {

	if proposedBlock == nil {
		return nil
	}

	proposal := createProposal(proposedBlock)
	signProposal(proposal, ba.privateKey)

	ba.BroadcastProposal(*proposal)

	ba.log.Printf("Proposal broadcasted for block: %s \n", ByteToBase64String(proposal.BlockHash))

	return proposal
}

// calculateSeed calculates seed field of a block
func (ba *BAStar) calculateSeed(block *blockchain.Block, previousBlock *blockchain.Block) {
	vrfInput := fmt.Sprintf("%s|%d", previousBlock.SeedHash, block.Index)
	block.SeedHash, block.SeedProof = ba.sortition.vrf.ProduceProof([]byte(vrfInput))
}

func (ba *BAStar) createEmptyBlock() {
	previousBlock := ba.blockchain.GetLastBlock()
	round := ba.blockchain.GetBlockHeight()
	ba.emptyBlock = ba.memoryPool.CreateEmptyBlock(previousBlock, round)
	ba.log.Printf("Empty block created %s \n", ByteToBase64String(ba.emptyBlock.Hash()))
}

func (ba *BAStar) validateBlock(block *blockchain.Block) bool {

	if isBlockSignatureValid(block) == false {
		ba.log.Println("WARNING: Block signature is not valid")
		return false
	}

	lastBlock := ba.blockchain.GetLastBlock()
	lastBlockHash := ba.blockchain.GetLastBlockHash()
	if bytes.Equal(lastBlockHash, block.PrevHash) == false {
		ba.log.Println("WARNING: Block previous hash is not correct")
		return false
	}

	seed := string(lastBlock.SeedHash)
	threshold := ba.params.ThresholdProposer
	role := RoleProposer
	userMoney := ba.params.UserMoney
	totalMoney := ba.params.TotalMoney

	result := ba.sortition.Verify(block.Issuer, block.VrfHash, block.VrfProof, seed, threshold, role, userMoney, totalMoney)

	if result == 0 {
		ba.log.Println("WARNING: Block VRF block is not valid")
		return false
	}

	return true
}

func (ba *BAStar) validateVote(vote Vote, step string, threshold int) (numVotes uint64, value []byte, sortitionHash []byte) {

	if isVoteSignatureValid(&vote) == false {
		ba.log.Println("WARNING: Vote signature is not valid")
	}

	lastBlock := ba.blockchain.GetLastBlock()
	lastBlockHash := ba.blockchain.GetLastBlockHash()
	if bytes.Equal(lastBlockHash, vote.LastBlockHash) == false {
		ba.log.Printf("WARNING: Vote previous hash is not correct. Round:%d  %s != %s\n", vote.Round, ByteToBase64String(lastBlockHash), ByteToBase64String(vote.LastBlockHash))
		return
	}

	round := ba.blockchain.GetBlockHeight()
	seed := string(lastBlock.SeedHash)
	role := fmt.Sprintf("%s|%d|%s", RoleCommittee, round, step)
	userMoney := ba.params.UserMoney
	totalMoney := ba.params.TotalMoney

	selectionCount := ba.sortition.Verify(vote.Issuer, vote.VrfHash, vote.VrfProof, seed, threshold, role, userMoney, totalMoney)

	if selectionCount != vote.VoteCount {
		ba.log.Printf("ERROR: vote count is not correct. Round: %d Step: %s \n", vote.Round, vote.Step)
		return
	}

	return vote.VoteCount, vote.SelectedBlock, vote.VrfHash
}
