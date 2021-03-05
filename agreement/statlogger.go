package agreement

import (
	"fmt"
	"log"
	"os"
	"time"
)

type statData struct {
	round               int
	startTime           time.Time
	blockReceived       float64
	endOfBAWithoutFinal float64
	endOfBAWithFinal    float64

	localBlockAppended    bool
	finalConsensusReached bool
	consensusOnEmptyBlock bool

	blockHash string

	payloadSize int
}

// StatLogger defines a custom logger to log the important events
type StatLogger struct {
	data                statData
	delimeter           string
	log                 *log.Logger
	macroBlockSize      int
	concurrencyConstant int
}

// NewStatLogger creates a new StatLogger
func NewStatLogger(nodeID string, macroBlockSize int, concurrencyConstant int) *StatLogger {
	flags := log.Ldate | log.Ltime | log.Lmsgprefix
	statLogger := StatLogger{}
	statLogger.data.round = -1
	statLogger.log = log.New(os.Stderr, fmt.Sprintf("[stats]\t%s\t%d\t%d", nodeID, macroBlockSize, concurrencyConstant), flags)
	statLogger.delimeter = "\t"

	statLogger.macroBlockSize = macroBlockSize
	statLogger.concurrencyConstant = concurrencyConstant

	return &statLogger
}

// RoundStarted prints the log of the previous round and resets logger for the new round
func (sl *StatLogger) RoundStarted(round int) {

	if sl.data.round > -1 {
		sl.printLine()
	}

	sl.data.round = round
	sl.data.startTime = time.Now()
}

// BlockReceived marks the time of block receive event for the corresponding round
func (sl *StatLogger) BlockReceived(localBlockAppended bool) {
	sl.data.blockReceived = time.Since(sl.data.startTime).Seconds()
	sl.data.localBlockAppended = localBlockAppended
}

// SetAppendedPayloadSize sets appended paylaod size
func (sl *StatLogger) SetAppendedPayloadSize(payloadSize int) {
	sl.data.payloadSize = payloadSize
}

// EndOfBAWithoutFinalStep marks the end of BA without final event
func (sl *StatLogger) EndOfBAWithoutFinalStep() {
	sl.data.endOfBAWithoutFinal = time.Since(sl.data.startTime).Seconds()
}

// EndOfBAWithFinal marks the end of BA with fibal step
func (sl *StatLogger) EndOfBAWithFinal(finalConsensusReached bool, consensusOnEmptyBlock bool, blockHash []byte) {
	sl.data.endOfBAWithFinal = time.Since(sl.data.startTime).Seconds()
	sl.data.finalConsensusReached = finalConsensusReached
	sl.data.consensusOnEmptyBlock = consensusOnEmptyBlock
	if len(blockHash) > 0 {
		sl.data.blockHash = ByteToBase64String(blockHash[0:10])
	} else {
		sl.data.blockHash = "EMPTY_BLOCK"
	}
}

func (sl *StatLogger) printLine() {

	delimeter := sl.delimeter

	// appendes a tab and round
	line := fmt.Sprintf("%s%d", delimeter, sl.data.round)

	// appends block time
	if sl.data.localBlockAppended {
		line = fmt.Sprintf("%s%s%f", line, delimeter, 0.0)
	} else {
		line = fmt.Sprintf("%s%s%f", line, delimeter, sl.data.blockReceived)
	}

	// appends BA time without final step
	line = fmt.Sprintf("%s%s%f", line, delimeter, sl.data.endOfBAWithoutFinal)

	// appends BA time with final step
	line = fmt.Sprintf("%s%s%f", line, delimeter, sl.data.endOfBAWithFinal)

	// appends consensus Type
	if sl.data.finalConsensusReached {
		line = fmt.Sprintf("%s%s%s", line, delimeter, "FINAL")
	} else {
		line = fmt.Sprintf("%s%s%s", line, delimeter, "TENTATIVE")
	}

	// appends block Type
	if sl.data.consensusOnEmptyBlock {
		line = fmt.Sprintf("%s%s%s", line, delimeter, "EMPTY_BLOCK")
	} else {
		line = fmt.Sprintf("%s%s%s", line, delimeter, "PROPOSED_BLOCK")
	}

	// appends block hash first 8 character encoded in base64
	line = fmt.Sprintf("%s%s%s", line, delimeter, sl.data.blockHash)

	// appends payload size
	line = fmt.Sprintf("%s%s%d", line, delimeter, sl.data.payloadSize)

	sl.log.Println(line)
}
