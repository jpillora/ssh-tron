package tron

import "log"

const (
	full = "⣿"
	top  = "⠛"
	bot  = "⣤"
	none = " "
)

const (
	wall  = ID(0xffff)
	blank = ID(0x0000)
)

type Board [][]ID

func NewBoard(width, height uint8) Board {
	if height%2 != 0 {
		log.Fatal("Height must be even")
	}
	board := make([][]ID, width)
	for w := uint8(0); w < width; w++ {
		board[w] = make([]ID, height)
		for h := uint8(0); h < height; h++ {
			board[w][h] = blank
		}
	}
	return board
}

func (b Board) width() int {
	return len(b)
}

func (b Board) height() int {
	return len(b[0])
}

func (b Board) termHeight() int {
	return b.height() / 2
}
