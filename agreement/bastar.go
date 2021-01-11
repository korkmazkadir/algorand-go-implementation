package agreement

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"../blockchain"
	"../config"
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
	emptyBlock *blockchain.MacroBlock

	sortition *sortition

	params           config.ProtocolParams
	validationParams config.ValidationParameters

	context
	wg *sync.WaitGroup

	stopOnRound int

	log        *log.Logger
	statLogger *StatLogger

	messageCountLogger
}

// NewBAStar creates an instance of agrreement protocol
func NewBAStar(params config.ProtocolParams, validationParams config.ValidationParameters, publicKey []byte, privateKey []byte, memoryPool blockchain.MemoryPool, blockchain blockchain.Blockchain, logger *log.Logger, stopOnRound int) *BAStar {
	ba := new(BAStar)
	ba.networkReadySig = make(chan struct{}, 1)

	//Creates a paylaod codec
	ba.demultiplexer = newDemux(1)

	ba.outgoingMessages = make(chan node.Message, 100)

	ba.params = params
	ba.validationParams = validationParams
	ba.publickKey = publicKey
	ba.privateKey = privateKey
	ba.memoryPool = memoryPool
	ba.blockchain = blockchain
	ba.sortition = newSortition(ba.privateKey)

	ba.wg = &sync.WaitGroup{}

	ba.log = logger
	ba.statLogger = NewStatLogger()

	ba.stopOnRound = stopOnRound

	ba.messageCountLogger.round = 1

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

	for {

		currentRound := ba.blockchain.GetBlockHeight()
		ba.statLogger.RoundStarted(currentRound)

		if ba.stopOnRound == currentRound {
			ba.log.Printf("Target round %d is reached. Node will exit after sleeping 1 minute\n", currentRound)
			time.Sleep(1 * time.Minute)
			os.Exit(0)
		}

		//creates an empty block for the current round to use
		ba.createEmptyBlock()

		ba.log.Printf("Last Block Hash %s \n", ByteToBase64String(ba.blockchain.GetLastBlockHash()))

		round := ba.blockchain.GetBlockHeight()

		proposedBlock := ba.proposeBlock()

		localProposal := ba.submitProposal(proposedBlock)

		selections, highestPriorityBlocks := ba.waitForProposals(localProposal, proposedBlock)
		ba.statLogger.BlockReceived(false)

		var selection SelectionVector

		if selections.Size() == 0 {
			selection = NewSelectionVector(ba.emptyBlock.Hash())
		} else {
			selection = selections
		}

		selection = ba.reduction(round, selection)
		ba.log.Printf("Result of reduction: %s\n", ByteToBase64String(selection.Hash()))

		selection = *ba.binaryBA(round, selection)
		ba.log.Printf("Result of binary BA: %s\n", ByteToBase64String(selection.Hash()))

		ba.statLogger.EndOfBAWithoutFinalStep()

		r, _ := ba.countVotes(round, StepFinal, ba.params.TSmallFinal, ba.params.TBigFinal, ba.params.LamdaStep)

		if bytes.Equal(r.Hash(), selection.Hash()) {

			ba.log.Printf("FINAL CONSENSUS on %s\n", ByteToBase64String(selection.Hash()))

			var blockToAppend *blockchain.MacroBlock
			//if highestPriorityBlock != nil && bytes.Equal(highestPriorityBlock.Hash(), blockHash) {

			missingBlocks := GetMissingBlocks(selection, highestPriorityBlocks)
			if missingBlocks == nil {

				blockToAppend = ConstructMacroBlock(selection, highestPriorityBlocks)
				ba.statLogger.EndOfBAWithFinal(true, false, blockToAppend.Hash())

			} else if bytes.Equal(ba.emptyBlock.Hash(), selection.Hash()) {

				ba.statLogger.EndOfBAWithFinal(true, true, ba.emptyBlock.Hash())
				ba.log.Println("Appending empty block to the blockchain!!")
				blockToAppend = ba.emptyBlock

			} else {

				ba.statLogger.EndOfBAWithFinal(true, false, selection.Hash())
				receivedBlocks := ba.waitForMissingBlock(round, missingBlocks)
				highestPriorityBlocks = append(highestPriorityBlocks, receivedBlocks...)
				blockToAppend = ConstructMacroBlock(selection, highestPriorityBlocks)
				ba.statLogger.BlockReceived(false)

			}

			err := ba.blockchain.AppendBlock(*blockToAppend)
			if err != nil {
				panic(err)
			}

		} else {

			ba.log.Printf("TENTATIVE CONSENSUS on %s\n", ByteToBase64String(selection.Hash()))

			var blockToAppend *blockchain.MacroBlock
			missingBlocks := GetMissingBlocks(selection, highestPriorityBlocks)

			if missingBlocks == nil {

				blockToAppend = ConstructMacroBlock(selection, highestPriorityBlocks)
				ba.statLogger.EndOfBAWithFinal(false, false, blockToAppend.Hash())

			} else if bytes.Equal(ba.emptyBlock.Hash(), selection.Hash()) {

				ba.statLogger.EndOfBAWithFinal(false, true, ba.emptyBlock.Hash())
				ba.log.Println("Appending empty block to the blockchain!!")
				blockToAppend = ba.emptyBlock

			} else {

				ba.statLogger.EndOfBAWithFinal(false, false, selection.Hash())
				receivedBlocks := ba.waitForMissingBlock(round, missingBlocks)
				highestPriorityBlocks = append(highestPriorityBlocks, receivedBlocks...)
				blockToAppend = ConstructMacroBlock(selection, highestPriorityBlocks)
				ba.statLogger.BlockReceived(false)

			}

			err := ba.blockchain.AppendBlock(*blockToAppend)
			if err != nil {
				panic(err)
			}

		}

		//Sets round on demux
		ba.demultiplexer.SetRound(ba.blockchain.GetBlockHeight())
		ba.messageCountLogger.setRound(ba.blockchain.GetBlockHeight())

		//Sets payload size of appended block for logging purposes
		payloadSize := ba.blockchain.GetLastBlock().PayloadSize()
		ba.statLogger.SetAppendedPayloadSize(payloadSize)

	}
}

func (ba *BAStar) waitForMissingBlock(round int, blockHashes [][]byte) []blockchain.Block {

	start := time.Now()

	for _, blockHash := range blockHashes {
		ba.log.Printf("Waiting for missing block %s\n", ByteToBase64String(blockHash))

		if bytes.Equal(blockHash, ba.emptyBlock.Hash()) {
			//panic("Waiting for empty block!!!")
			ba.log.Println("WARNING: returning empty block to append the chain. Be careful!!!")
			panic("waiting for empty block!!!")
			//return ba.emptyBlock
		}
	}

	blockChan := ba.demultiplexer.GetBlockChan(round)

	receivedBlocks := []blockchain.Block{}

	numberOfBlocksToWait := len(blockHashes)
	for len(receivedBlocks) != numberOfBlocksToWait {

		select {
		case incommingBlock := <-blockChan:

			block := incommingBlock.block
			forwardBlock := incommingBlock.forward

			if ba.validateBlock(&block) == false {
				ba.log.Printf("WARNING: An invalid block received %s \n", ByteToBase64String(block.Hash()))
				continue
			}

			for _, blockHash := range blockHashes {

				if bytes.Equal(block.Hash(), blockHash) {
					ba.messageCountLogger.forwardingBlock()
					forwardBlock()
					ba.log.Printf("Missing block received %s Time elpased: %f \n", ByteToBase64String(blockHash), time.Since(start).Seconds())
					receivedBlocks = append(receivedBlocks, block)
				}

			}

			//ba.log.Printf("Discarting block %s round: %d \n", ByteToBase64String(block.Hash()), block.Index)

		}
	}

	return receivedBlocks
}

//TODO: Update name of the function: waitForProposalsAndBlocks
func (ba *BAStar) waitForProposals(localProposal *Proposal, localBlock *blockchain.Block) (SelectionVector, []blockchain.Block) {

	ba.log.Printf("Waiting for proposals and blocks...")

	sleepTime := time.Duration(ba.params.LamdaPriority + ba.params.LamdaStepVar)
	timeout := time.After(sleepTime * time.Second)

	proposalChan := ba.demultiplexer.GetProposalChan(ba.blockchain.GetBlockHeight())
	blockChan := ba.demultiplexer.GetBlockChan(ba.blockchain.GetBlockHeight())

	receivedBlocks := []blockchain.Block{}
	selectionVector := NewSelectionVectorWithSize(ba.params.ConcurrencyConstant)

	if localProposal != nil {
		receivedBlocks = append(receivedBlocks, *localBlock)
		selectionVector.Add(*localProposal)
	}

	for {
		select {
		case incommingProposal := <-proposalChan:

			proposal := incommingProposal.proposal
			forwardProposal := incommingProposal.forward

			if ba.validateProposal(&proposal) == false {
				ba.log.Printf("WARNING: An invalid proposal received %s \n", ByteToBase64String(proposal.Hash()))
				continue
			}

			//forwards the proposal
			if selectionVector.Add(proposal) {
				ba.log.Printf("Forwarding proposal for the block: %s\n", ByteToBase64String(proposal.BlockHash))
				ba.messageCountLogger.forwardingProposal()
				forwardProposal()
			} else {
				ba.log.Printf("Proposal Not forwarded for the block %s\n", ByteToBase64String(proposal.BlockHash))
			}

		case incommingBlock := <-blockChan:

			block := incommingBlock.block
			forwardBlock := incommingBlock.forward

			if ba.validateBlock(&block) == false {
				ba.log.Printf("WARNING: An invalid block received %s \n", ByteToBase64String(block.Hash()))
				continue
			}

			// adds the received block to the received block list
			receivedBlocks = append(receivedBlocks, block)

			localProposal := createProposal(&block)
			if selectionVector.Add(*localProposal) {
				ba.log.Printf("Forwarding block %s\n", ByteToBase64String(block.Hash()))
				ba.messageCountLogger.forwardingBlock()
				forwardBlock()
			} else {
				ba.log.Printf("Block not forwarded %s\n", ByteToBase64String(block.Hash()))
			}

		case <-timeout:

			if selectionVector.Size() == 0 {
				ba.log.Println("No proposal received!!!")
			} else {
				missingBlocks := GetMissingBlocks(selectionVector, receivedBlocks)
				ba.log.Printf("Highest priority proposal for the block %s  Block received %t \n", ByteToBase64String(selectionVector.Hash()), missingBlocks == nil)
			}

			return selectionVector, receivedBlocks
		}
	}

}

func (ba *BAStar) reduction(round int, sv SelectionVector) SelectionVector {

	stepThreshold := ba.params.TSmallStep
	ba.committeeVote(round, StepReductionOne, stepThreshold, sv)

	voteThreshold := ba.params.TBigStep
	timerValueOne := ba.params.LamdaBlock + ba.params.LamdaStep

	blockHash1, isTimerExpired := ba.countVotes(round, StepReductionOne, voteThreshold, stepThreshold, timerValueOne)
	emptyHash := ba.emptyBlock.Hash()

	if isTimerExpired {
		ba.committeeVote(round, StepReductionTwo, stepThreshold, NewSelectionVector(emptyHash))
	} else {
		ba.committeeVote(round, StepReductionTwo, stepThreshold, blockHash1)
	}

	blockHash2, isTimerExpired := ba.countVotes(round, StepReductionTwo, voteThreshold, stepThreshold, ba.params.LamdaStep)

	if isTimerExpired {
		return NewSelectionVector(emptyHash)
	}

	return blockHash2
}

func (ba *BAStar) committeeVote(round int, step string, stepThreshold int, selectionVector SelectionVector) {

	lastBlock := ba.blockchain.GetLastBlock()
	seed := lastBlock.SeedHash()

	role := fmt.Sprintf("%s|%d|%s", RoleCommittee, round, step)
	userMoney := ba.params.UserMoney
	totalMoney := ba.params.TotalMoney

	hash, proof, numberOfTimesSelected := ba.sortition.Select(string(seed), stepThreshold, role, userMoney, totalMoney)

	if numberOfTimesSelected == 0 {
		ba.localVote = nil
		return
	}

	vote := Vote{
		Issuer:          ba.publickKey,
		VrfHash:         hash,
		VrfProof:        proof,
		VoteCount:       numberOfTimesSelected,
		Round:           round,
		Step:            step,
		LastBlockHash:   ba.blockchain.GetLastBlockHash(),
		SelectionVector: selectionVector,
	}

	//signs vote
	signVote(&vote, ba.privateKey)

	ba.log.Printf("Voting for step: %s block: %s \n", step, ByteToBase64String(vote.SelectionVector.Hash()))
	ba.messageCountLogger.forwardingVote()
	ba.BroadcastVote(vote)

	ba.localVote = &vote
}

func (ba *BAStar) countVotes(round int, step string, voteThreshold float32, stepThreshold int, timeout int) (selectionVector SelectionVector, isTimerExprired bool) {

	counts := make(map[string]uint64)
	voters := make(map[string][]byte)

	sleepTime := time.Duration(timeout)
	timer := time.After(sleepTime * time.Second)
	incommingVotes := ba.demultiplexer.GetVoteChan(round, step)

	if ba.localVote != nil {
		voters[string(ba.localVote.Issuer)] = ba.localVote.Issuer
		counts[string(ba.localVote.SelectionVector.Hash())] = ba.localVote.VoteCount
		ba.log.Printf("Local Vote for %s --> count:%d\n", ByteToBase64String(ba.localVote.SelectionVector.Hash()), ba.localVote.VoteCount)
	}

	for {
		select {
		case incommingVote := <-incommingVotes:

			vote := incommingVote.vote
			forwardCallback := incommingVote.forward

			numVotes, selectionVector, _ := ba.validateVote(vote, step, stepThreshold)

			if numVotes == 0 {
				continue
			}

			if voters[string(vote.Issuer)] != nil || numVotes == 0 {
				continue
			}

			voters[string(vote.Issuer)] = vote.Issuer
			counts[string(selectionVector.Hash())] = counts[string(selectionVector.Hash())] + numVotes

			//ba.log.Printf("Vote for %s --> count:%d\n", ByteToBase64String(selectedBlockHash), numVotes)

			ba.messageCountLogger.forwardingVote()
			forwardCallback()
			//ba.log.Println("Vote forwarded!")

			//TODO: Check this line, ceil float64 does not seems good
			if float64(counts[string(selectionVector.Hash())]) > math.Ceil(float64(voteThreshold)*float64(stepThreshold)) {
				ba.log.Printf("A block has reached the target vote count: %s\n", ByteToBase64String(selectionVector.Hash()))
				ba.printVoteCount(counts)
				return selectionVector, false
			}

		case <-timer:
			ba.log.Println("Timer expired for count votes.")
			ba.printVoteCount(counts)
			//WARNING it was returning nil!!!!!!
			return selectionVector, true
		}
	}

}

func (ba *BAStar) printVoteCount(voteCountMap map[string]uint64) {

	for blockHash, count := range voteCountMap {
		ba.log.Printf("[vote-count] Block: %s --> Votes: %d \n", ByteToBase64String([]byte(blockHash)), count)
	}

}

func (ba *BAStar) binaryBA(round int, selectionVector SelectionVector) *SelectionVector {

	step := 1
	r := selectionVector
	emptyBlockHash := ba.emptyBlock.Hash()

	// TODO define a maxstep
	for step < 255 {

		ba.log.Printf("=======> Binary BA STEP %d \n", step)

		ba.committeeVote(round, strconv.Itoa(step), ba.params.TSmallStep, r)
		r, timerExpired := ba.countVotes(round, strconv.Itoa(step), ba.params.TBigStep, ba.params.TSmallStep, ba.params.LamdaStep)

		if timerExpired {
			r = selectionVector
		} else if bytes.Equal(emptyBlockHash, r.Hash()) == false {

			// TODO: I have removed this. Consider to open later
			//votes for the same block for next 3 rounds
			//for i := 1; i < 4; i++ {
			//	ba.committeeVote(round, strconv.Itoa(step+1), ba.params.TSmallStep, r)
			//}

			if step == 1 {
				ba.committeeVote(round, StepFinal, ba.params.TBigFinal, r)
			}

			return &r
		}

		/***************************************/
		step++
		ba.log.Printf("=======> Binary BA STEP %d \n", step)
		ba.committeeVote(round, strconv.Itoa(step), ba.params.TSmallStep, r)
		r, timerExpired = ba.countVotes(round, strconv.Itoa(step), ba.params.TBigStep, ba.params.TSmallStep, ba.params.LamdaStep)
		if timerExpired {
			r = NewSelectionVector(emptyBlockHash)
		} else if r.SelectsEmptyBlock(emptyBlockHash) {
			return &r
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
	ba.calculateSeed(block, ba.blockchain.GetLastBlockSeedHash())

	//signs block
	signBlock(block, ba.privateKey)

	//broadcasts the block
	//TODO: decide between using pointer vs value
	//ba.BroadcastBlock(*block)

	ba.log.Printf("Proposed block: %s (Not Broadcasted!!!)\n", ByteToBase64String(block.Hash()))

	return block
}

//TODO: update name as submitProposalAndBlock
func (ba *BAStar) submitProposal(proposedBlock *blockchain.Block) *Proposal {

	if proposedBlock == nil {
		return nil
	}

	proposal := createProposal(proposedBlock)
	signProposal(proposal, ba.privateKey)

	//Submits proposal first
	ba.messageCountLogger.forwardingProposal()
	ba.BroadcastProposal(*proposal)

	//submits a block without waiting
	ba.messageCountLogger.forwardingBlock()
	ba.BroadcastBlock(*proposedBlock)

	ba.log.Printf("Proposal and the block broadcasted: %s \n", ByteToBase64String(proposal.BlockHash))

	return proposal
}

// calculateSeed calculates seed field of a block
func (ba *BAStar) calculateSeed(block *blockchain.Block, previousBlockSeedHash []byte) {
	vrfInput := fmt.Sprintf("%s|%d", previousBlockSeedHash, block.Index)
	block.SeedHash, block.SeedProof = ba.sortition.vrf.ProduceProof([]byte(vrfInput))
}

func (ba *BAStar) createEmptyBlock() {
	previousBlock := ba.blockchain.GetLastBlock()
	round := ba.blockchain.GetBlockHeight()
	roundEmptyBlock := ba.memoryPool.CreateEmptyBlock(previousBlock, round)
	ba.emptyBlock = blockchain.NewMacroBlock([]blockchain.Block{*roundEmptyBlock})

	ba.log.Printf("Empty block created %s \n", ByteToBase64String(ba.emptyBlock.Hash()))
}

func (ba *BAStar) validateProposal(proposal *Proposal) bool {

	lastBlock := ba.blockchain.GetLastBlock()
	lastBlockHash := ba.blockchain.GetLastBlockHash()
	if bytes.Equal(lastBlockHash, proposal.PrevHash) == false {
		ba.log.Println("WARNING: Block previous hash is not correct")
		return false
	}

	if ba.validationParams.ValidateBlock == false {
		//ba.log.Println("WARNING: Did not validate signature and VRF of the block because validation is disabled.")
		return true
	}

	if isProposalSignatureValid(proposal) == false {
		ba.log.Println("WARNING: Proposal signature is not valid")
		return false
	}

	seed := string(lastBlock.SeedHash())
	threshold := ba.params.ThresholdProposer
	role := RoleProposer
	userMoney := ba.params.UserMoney
	totalMoney := ba.params.TotalMoney

	//Locally calculates the vrf hash
	//Should do it for blocks also
	vrfHash := digest(proposal.VrfProof)

	result := ba.sortition.Verify(proposal.Issuer, vrfHash, proposal.VrfProof, seed, threshold, role, userMoney, totalMoney)

	if result == 0 {
		ba.log.Println("WARNING: Proposal VRF block is not valid")
		return false
	}

	return true

}

func (ba *BAStar) validateBlock(block *blockchain.Block) bool {

	lastBlock := ba.blockchain.GetLastBlock()
	lastBlockHash := ba.blockchain.GetLastBlockHash()
	if bytes.Equal(lastBlockHash, block.PrevHash) == false {
		ba.log.Println("WARNING: Block previous hash is not correct")
		return false
	}

	if ba.validationParams.ValidateBlock == false {
		//ba.log.Println("WARNING: Did not validate signature and VRF of the block because validation is disabled.")
		return true
	}

	if isBlockSignatureValid(block) == false {
		ba.log.Println("WARNING: Block signature is not valid")
		return false
	}

	seed := string(lastBlock.SeedHash())
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

func (ba *BAStar) validateVote(vote Vote, step string, threshold int) (numVotes uint64, selectionVector SelectionVector, sortitionHash []byte) {

	lastBlock := ba.blockchain.GetLastBlock()
	lastBlockHash := ba.blockchain.GetLastBlockHash()
	if bytes.Equal(lastBlockHash, vote.LastBlockHash) == false {
		ba.log.Printf("WARNING: Vote previous hash is not correct. Round:%d  %s != %s\n", vote.Round, ByteToBase64String(lastBlockHash), ByteToBase64String(vote.LastBlockHash))
		return
	}

	if ba.validationParams.ValidateVote == false {
		//ba.log.Println("WARNING: Did not validate signature and VRF of the vote because validation is disabled.")
		return vote.VoteCount, vote.SelectionVector, vote.VrfHash
	}

	if isVoteSignatureValid(&vote) == false {
		ba.log.Println("WARNING: Vote signature is not valid")
	}

	round := ba.blockchain.GetBlockHeight()
	seed := string(lastBlock.SeedHash())
	role := fmt.Sprintf("%s|%d|%s", RoleCommittee, round, step)
	userMoney := ba.params.UserMoney
	totalMoney := ba.params.TotalMoney

	selectionCount := ba.sortition.Verify(vote.Issuer, vote.VrfHash, vote.VrfProof, seed, threshold, role, userMoney, totalMoney)

	if selectionCount != vote.VoteCount {
		ba.log.Printf("ERROR: vote count is not correct. Round: %d Step: %s \n", vote.Round, vote.Step)
		return
	}

	return vote.VoteCount, vote.SelectionVector, vote.VrfHash
}
