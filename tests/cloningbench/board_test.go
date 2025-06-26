package cloningbench

import (
	"testing"
)

type Bitboard uint64

type Board struct {
	Pawns, Knights, Bishops, Rooks, Queens, Kings [2]Bitboard
	Occupied                                      [2]Bitboard
	Hash                                          uint64
	EnPassantSquare                               int8
	Flags                                         uint8
}

// Value receiver
func (b Board) CloneValue() Board {
	return b
}

// Pointer receiver
func (b *Board) ClonePointer() Board {
	return *b
}

var board = Board{
	Hash: 12345,
}

// Benchmarks
func BenchmarkCloneValue(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = board.CloneValue()
	}
}

func BenchmarkClonePointer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = (&board).ClonePointer()
	}
}
