package agreement

import (
	"crypto/ed25519"
	"fmt"
	"testing"
	"time"
)

func TestSortition(t *testing.T) {

	pk, sk, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Errorf("could not generate key %s", err)
	}

	s := newSortition(sk)

	seed := "hello-there"
	threshold := 26
	role := "leader"
	userMoney := uint64(10)
	totalMoney := uint64(100)

	start := time.Now()

	_, proof, selectedTimes := s.Select(seed, threshold, role, userMoney, totalMoney)

	fmt.Println("select elapsed time: ", time.Since(start))

	start = time.Now()
	result := s.Verify(pk, proof, seed, threshold, role, userMoney, totalMoney)
	fmt.Println("verify elapsed time: ", time.Since(start))

	fmt.Printf("selected times: %d result: %d \n", selectedTimes, result)

	if result != selectedTimes {
		t.Errorf("could not verify the result: %d != %d", result, selectedTimes)
	}

}
