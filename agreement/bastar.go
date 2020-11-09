package agreement

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"../blockchain"
	"../filter"
	"github.com/korkmazkadir/go-rpc-node/node"
)

// TODO: tag field of message can be a byte
const tagBlock = "B"
const tagVote = "V"

// BAStar implements algorand agreement protocol
type BAStar struct {
	id int
	applicationImp

	memoryPool blockchain.MemoryPool
	blockchain blockchain.Blockchain

	sortition *sortition
	params    ProtocolParams
	context
	wg  *sync.WaitGroup
	log *log.Logger
}

// NewBAStar creates an instance of agrreement protocol
func NewBAStar(params ProtocolParams, publicKey []byte, privateKey []byte, memoryPool blockchain.MemoryPool, blockchain blockchain.Blockchain, logger *log.Logger) *BAStar {
	ba := new(BAStar)
	ba.networkReadySig = make(chan struct{}, 1)

	// TTL 60 seconds
	ba.messageFilter = filter.NewUniqueMessageFilter(60)

	// TODO: buffer sizes must be considered carefully
	ba.incommingBlockChan = make(chan incommingBlock, 100)
	ba.incommingVoteChan = make(chan incommingVote, 100)
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

		time.Sleep(10 * time.Second)

		ba.log.Println("Started...")

		ba.wg.Add(1)
		ba.mainLoop()
	}()

}

func (ba *BAStar) mainLoop() {

	for {

		ba.log.Printf("---------> Last Block Hash %s \n", ba.blockchain.GetLastBlockHash())

		round := ba.blockchain.GetBlockHeight()

		proposedBlock := ba.proposeBlock()
		highestPriorityBlock := ba.waitForBlocks(proposedBlock)

		var blockHash []byte
		if highestPriorityBlock != nil {
			blockHash = highestPriorityBlock.Hash()
		}

		blockHash = ba.reduction(round, blockHash)
		ba.log.Printf("Result of reduction: %s\n", ByteToBase64String(blockHash))

		blockHash = ba.binaryBA(round, blockHash)
		ba.log.Printf("Result of binary BA: %s\n", ByteToBase64String(blockHash))

		r, _ := ba.countVotes(round, StepFinal, ba.params.TSmallFinal, ba.params.TBigFinal, ba.params.LamdaStep)
		if bytes.Equal(r, blockHash) {
			ba.log.Printf("FINAL CONSENSUS on %s\n", ByteToBase64String(blockHash))

			if bytes.Equal(blockHash, highestPriorityBlock.Hash()) {
				err := ba.blockchain.AppendBlock(*highestPriorityBlock)
				if err != nil {
					panic(err)
				}
			} else {
				ba.log.Printf("BLOCK NOT HERE!!!! %s\n", ByteToBase64String(blockHash))
			}

		} else {
			ba.log.Printf("TENTATIVE CONSENSUS on %s\n", ByteToBase64String(blockHash))
		}
	}
}

func (ba *BAStar) waitForBlocks(proposedBlock *blockchain.Block) *blockchain.Block {

	ba.log.Printf("Waiting for proposals...")

	sleepTime := time.Duration(ba.params.LamdaPriority + ba.params.LamdaStepVar)
	timeout := time.After(sleepTime * time.Second)

	var highestPriorityBlock = proposedBlock

	for {

		select {
		case incommingBlock := <-ba.incommingBlockChan:

			block := incommingBlock.block
			forwardBlock := incommingBlock.forward

			//TODO validate the block
			//ba.log.Printf("Block received: %s\n", ByteToBase64String(block.Hash()))
			if highestPriorityBlock == nil || compareBlocks(&block, highestPriorityBlock) == 1 {
				ba.log.Printf("Block forwarded: %s\n", ByteToBase64String(block.Hash()))
				highestPriorityBlock = &block
				forwardBlock()
			} else {
				ba.log.Printf("Not forwarded: %s\n", ByteToBase64String(block.Hash()))
			}

		case <-timeout:

			if highestPriorityBlock != nil {
				ba.log.Printf("Highest priority block: %s \n", ByteToBase64String(highestPriorityBlock.Hash()))
			} else {
				ba.log.Println("Highest priority block: nil")
			}

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
	emptyHash := emptyHash(round, ba.blockchain.GetLastBlockHash())

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
		return
	}

	vote := Vote{
		SenderPK:      ba.publickKey,
		VrfHash:       hash,
		VrfProof:      proof,
		VoteCount:     numberOfTimesSelected,
		Round:         round,
		Step:          step,
		LastBlockHash: ba.blockchain.GetLastBlockHash(),
		//LastBlockHash: lastBlock.Hash(),
		SelectedBlock: blockHash,
	}

	ba.log.Printf("Voting for step: %s block: %s \n", step, ByteToBase64String(vote.SelectedBlock))
	ba.BroadcastVote(vote)
}

func (ba *BAStar) countVotes(round int, step string, voteThreshold float32, stepThreshold int, timeout int) (blockHash []byte, isTimerExprired bool) {

	counts := make(map[string]int)
	voters := make(map[string][]byte)

	sleepTime := time.Duration(timeout)
	timer := time.After(sleepTime * time.Second)
	incommingVotes := ba.incommingVoteChan

	for {
		select {
		case incommingVote := <-incommingVotes:

			vote := incommingVote.vote
			forwardCallback := incommingVote.forward

			numVotes, selectedBlockHash, _ := ba.validateVote(vote)

			if voters[string(vote.SenderPK)] != nil || numVotes == 0 {
				continue
			}

			voters[string(vote.SenderPK)] = vote.SenderPK
			counts[string(selectedBlockHash)] = counts[string(selectedBlockHash)] + numVotes

			ba.log.Printf("Vote for %s --> count:%d\n", ByteToBase64String(selectedBlockHash), numVotes)

			forwardCallback()

			//TODO: Check this line, ceil float64 does not seems good
			if counts[string(selectedBlockHash)] > int(math.Ceil(float64(voteThreshold)*float64(stepThreshold))) {
				ba.log.Printf("A block has reached the target vote count: %s\n", ByteToBase64String(selectedBlockHash))
				return selectedBlockHash, false
			}

		case <-timer:
			ba.log.Println("Timer expired for count votes.")

			return nil, true
		}
	}

}

func (ba *BAStar) validateVote(vote Vote) (numVotes int, value []byte, sortitionHash []byte) {

	//TODO: Validate Signature
	//TODO: validate Sortition

	numVotes = 0
	value = nil
	sortitionHash = nil

	lastBlockHash := ba.blockchain.GetLastBlockHash()
	if bytes.Equal(lastBlockHash, vote.LastBlockHash) == false {
		ba.log.Printf("Votes previous block is not correct %s not equals %s \n", ByteToBase64String(lastBlockHash), ByteToBase64String(vote.LastBlockHash))
		return
	}

	return vote.VoteCount, vote.SelectedBlock, vote.VrfHash
}

func (ba *BAStar) binaryBA(round int, blockHash []byte) []byte {

	lastBlock := ba.blockchain.GetLastBlock()

	step := 1
	r := blockHash
	emptyBlockHash := emptyHash(round, lastBlock.Hash())
	// TODO define a maxstep
	for step < 255 {

		ba.committeeVote(round, strconv.Itoa(step), ba.params.TSmallStep, r)
		r, timerExpired := ba.countVotes(round, strconv.Itoa(step), ba.params.TBigStep, ba.params.TSmallStep, ba.params.LamdaStep)

		if timerExpired {
			r = blockHash
		} else if bytes.Equal(emptyBlockHash, r) == false {

			//votes for the same block for next 3 rounds
			for i := 1; i < 4; i++ {
				ba.committeeVote(round, strconv.Itoa(step+1), ba.params.TSmallStep, r)
			}

			if step == 1 {
				ba.committeeVote(round, StepFinal, ba.params.TBigFinal, r)
			}

			return r
		}

		/***************************************/

		step++
		ba.committeeVote(round, strconv.Itoa(step), ba.params.TSmallStep, r)
		r, timerExpired = ba.countVotes(round, strconv.Itoa(step), ba.params.TBigStep, ba.params.TSmallStep, ba.params.LamdaStep)
		if timerExpired {
			r = emptyBlockHash
		} else if bytes.Equal(r, emptyBlockHash) {

			for i := 1; i < 4; i++ {
				ba.committeeVote(round, strconv.Itoa(step+1), ba.params.TSmallStep, r)
			}
			return r
		}

		/***************************************/

		step++
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
	ba.signBlock(block)

	//broadcasts the block
	//TODO: decide between using pointer vs value
	ba.BroadcastBlock(*block)

	ba.log.Printf("Proposed block: %s\n", ByteToBase64String(block.Hash()))

	return block
}

// calculateSeed calculates seed field of a block
func (ba *BAStar) calculateSeed(block *blockchain.Block, previousBlock *blockchain.Block) {
	vrfInput := fmt.Sprintf("%s|%d", previousBlock.SeedHash, block.Index)
	block.SeedHash, block.SeedProof = ba.sortition.vrf.ProduceProof([]byte(vrfInput))
}

func (ba *BAStar) signBlock(block *blockchain.Block) {
	blockHash := block.Hash()
	block.Signature = ed25519.Sign(ba.privateKey, blockHash)
	return
}
