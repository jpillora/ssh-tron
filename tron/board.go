package tron

import "errors"

const (
	wall  = ID(0xffff)
	blank = ID(0x0000)
)

// Board represents a board in a game.
// Each player has a board, which are used to send board deltas.
type Board [][]ID

// NewBoard returns an initialized Board.
func NewBoard(width, height uint8) (Board, error) {
	if height%2 != 0 {
		return nil, errors.New("height must be even")
	}
	if width%2 != 0 {
		return nil, errors.New("width must be even")
	}
	board := make([][]ID, width)
	for w := uint8(0); w < width; w++ {
		board[w] = make([]ID, height)
		for h := uint8(0); h < height; h++ {
			board[w][h] = blank
		}
	}
	return board, nil
}
