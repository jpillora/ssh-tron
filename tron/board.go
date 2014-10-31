package tron

import "log"

var (
	full = []byte("⣿")
	top  = []byte("⠛")
	bot  = []byte("⣤")
	none = []byte(" ")
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
	if width%2 != 0 {
		log.Fatal("Width must be even")
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

func (b Board) new() Board {
	return NewBoard(uint8(b.width()), uint8(b.height()))
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
