package agreement

import (
	"math/big"

	"gonum.org/v1/gonum/stat/distuv"
)

type sortition struct {
	vrf
}

func newSortition(privateKey []byte) *sortition {
	s := new(sortition)
	s.privateKey = privateKey
	return s
}

func (s *sortition) Select(seed string, threshold int, role string, userMoney uint64, totalMoney uint64) ([]byte, []byte, uint64) {

	input := seed + role
	vrfHash, proof := s.ProduceProof([]byte(input))

	ratio := calculateRatio(vrfHash)

	//fmt.Printf("[select] ratio: %f\n", ratio)

	numberOfTimesSelected := countSelected(threshold, userMoney, totalMoney, ratio)
	return vrfHash, proof, numberOfTimesSelected
}

func (s *sortition) Verify(publicKey []byte, proof []byte, seed string, threshold int, role string, userMoney uint64, totalMoney uint64) uint64 {

	input := seed + role
	if s.vrf.Verify(publicKey, []byte(input), proof) == false {
		//log.Println("could not verify vrf")
		return 0
	}

	calculatedHash := digest(proof)
	ratio := calculateRatio(calculatedHash)
	//fmt.Printf("[verify] ratio: %f\n", ratio)
	return countSelected(threshold, userMoney, totalMoney, ratio)
}

func countSelected(threshold int, userMoney uint64, totalMoney uint64, ratio float64) uint64 {

	p := float64(threshold) / float64(totalMoney)
	distribution := distuv.Binomial{N: float64(userMoney), P: p}

	var i uint64
	for i = 0; i < userMoney; i++ {

		boundary := distribution.CDF(float64(i))
		//fmt.Printf("\t[%d]\tboundary:%s\n", i, strconv.FormatFloat(boundary, 'f', -1, 64))

		if ratio <= boundary {
			return i
		}

	}

	return userMoney
}

func calculateRatio(vrfOutput []byte) float64 {
	/*********** From Algorand source code *******************/
	t := &big.Int{}
	t.SetBytes(vrfOutput)

	precision := uint(8 * (len(vrfOutput) + 1))
	max, b, err := big.ParseFloat("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0, precision, big.ToNearestEven)
	if b != 16 || err != nil {
		panic("failed to parse big float constant in sortition")
	}

	h := big.Float{}
	h.SetPrec(precision)
	h.SetInt(t)

	ratio := big.Float{}
	cratio, _ := ratio.Quo(&h, max).Float64()
	/************************************************************/
	return cratio
}
