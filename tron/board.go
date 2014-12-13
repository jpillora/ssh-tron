//the game has a board
//each player has a board,
//which are used to
//send board deltas
package tron

import "log"

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
