package chess

import (
	"math/rand"
)

func InitCardsWithDuplicates() ([]int8, []int8) {
	whiteCards := make([]int8, 5)
	blackCards := make([]int8, 5)

	for i := 0; i < 5; i++ {
		whiteCards[i] = int8(rand.Intn(6)) // 0..5
		blackCards[i] = int8(rand.Intn(6)) // 0..5
	}

	return whiteCards, blackCards
}
