package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
)

var boardConfig = []string{
	"=  '   =   '  =",
	" -   \"   \"   - ",
	"  -   ' '   -  ",
	"'  -   '   -  '",
	"    -     -    ",
	" \"   \"   \"   \" ",
	"  '   ' '   '  ",
	"=  '   *   '  =",
	"  '   ' '   '  ",
	" \"   \"   \"   \" ",
	"    -     -    ",
	"'  -   '   -  '",
	"  -   ' '   -  ",
	" -   \"   \"   - ",
	"=  '   =   '  =",
}

var tileSrc = map[byte][2]int{
	'A': {0, 0}, 'B': {0, 1}, 'C': {0, 2}, 'D': {0, 3}, 'E': {0, 4},
	'F': {0, 5}, 'G': {0, 6}, 'H': {0, 7}, 'I': {0, 8}, 'J': {0, 9},
	'K': {1, 0}, 'L': {1, 1}, 'M': {1, 2}, 'N': {1, 3}, 'O': {1, 4},
	'P': {1, 5}, 'Q': {1, 6}, 'R': {1, 7}, 'S': {1, 8}, 'T': {1, 9},
	'U': {2, 0}, 'V': {2, 1}, 'W': {2, 2}, 'X': {2, 3}, 'Y': {2, 4},
	'Z': {2, 5}, 'a': {2, 6}, 'b': {2, 7}, 'c': {2, 8}, 'd': {2, 9},
	'e': {3, 0}, 'f': {3, 1}, 'g': {3, 2}, 'h': {3, 3}, 'i': {3, 4},
	'j': {3, 5}, 'k': {3, 6}, 'l': {3, 7}, 'm': {3, 8}, 'n': {3, 9},
	'o': {4, 0}, 'p': {4, 1}, 'q': {4, 2}, 'r': {4, 3}, 's': {4, 4},
	't': {4, 5}, 'u': {4, 6}, 'v': {4, 7}, 'w': {4, 8}, 'x': {4, 9},
	'y': {5, 0}, 'z': {5, 1}, '?': {5, 2},
}

var boardSrc = map[byte][2]int{
	'-': {5, 3}, '=': {5, 4},
	'\'': {5, 5}, '"': {5, 6}, '*': {5, 7}, ' ': {5, 8},
}

// Doubled because of retina screen.
const squareDim = 2 * 34

func loadTilesImg() (image.Image, error) {
	tilesBytes, err := ioutil.ReadFile("tiles.png")
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(tilesBytes))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	expectedX := 10 * squareDim
	expectedY := 6 * squareDim
	if bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Dx() != expectedX || bounds.Dy() != expectedY {
		return nil, fmt.Errorf("unexpected size: %s vs %s", bounds.String(), image.Pt(expectedX, expectedY))
	}
	return img, nil
}

func drawBoard(tilesImg image.Image, boardConfig []string, board []string) (image.Image, error) {

	nRows := len(boardConfig)
	if nRows < 1 {
		return nil, fmt.Errorf("invalid boardConfig: expecting at least 1 row")
	}

	nCols := len(boardConfig[0])
	for i, row := range boardConfig {
		if len(row) != nCols {
			return nil, fmt.Errorf("invalid boardConfig: expecting row %d to have length %d", i+1, nCols)
		}
	}

	if nRows != len(board) {
		return nil, fmt.Errorf("invalid board: expecting %d rows", nRows)
	}

	for i, row := range board {
		if len(row) != nCols {
			return nil, fmt.Errorf("invalid board: expecting row %d to have length %d", i+1, nCols)
		}
	}

	// OK! They have the same dimensions.
	img := image.NewNRGBA(image.Rect(0, 0, nRows*squareDim, nCols*squareDim))

	// Draw the board.
	for r := 0; r < nRows; r++ {
		y := r * squareDim
		for c := 0; c < nCols; c++ {
			x := c * squareDim
			b := boardConfig[r][c]
			srcPt, ok := boardSrc[b]
			if !ok {
				srcPt = boardSrc[' ']
			}
			draw.Draw(img, image.Rect(x, y, x+squareDim, y+squareDim), tilesImg,
				image.Pt(srcPt[1]*squareDim, srcPt[0]*squareDim), draw.Over)
		}
	}

	// Draw the tiles.
	for r := 0; r < nRows; r++ {
		y := r * squareDim
		for c := 0; c < nCols; c++ {
			x := c * squareDim
			b := board[r][c]
			if b != ' ' {
				srcPt, ok := tileSrc[b]
				if !ok {
					srcPt = tileSrc['?']
				}
				draw.Draw(img, image.Rect(x, y, x+squareDim, y+squareDim), tilesImg,
					image.Pt(srcPt[1]*squareDim, srcPt[0]*squareDim), draw.Over)
			}
		}
	}

	return img, nil
}

func imgToPngBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func main() {
	// https://woogles.io/game/hBQhT94n
	var board = []string{
		" J DROGUE     C",
		" ARE WENT    GO",
		" GOX  r      LO",
		"  U   OM     AN",
		"  IVY NEB    I ",
		"PAl   T   B  R ",
		"EEL   IF  IT EL",
		"D EW  CENTS  SI",
		"   A   D  T   R",
		"  MI      AL QI",
		" FORK     TO O ",
		" AA I     EN P ",
		" Z  SHUN   E H ",
		"YE       VIRUS ",
		"AD             ",
	}

	// Cache this.
	tilesImg, err := loadTilesImg()
	if err != nil {
		panic(err)
	}

	boardImg, err := drawBoard(tilesImg, boardConfig, board)
	if err != nil {
		panic(err)
	}

	boardPngBytes, err := imgToPngBytes(boardImg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("writing %d bytes\n", len(boardPngBytes))

	err = ioutil.WriteFile("board.png", boardPngBytes, 0644)
	if err != nil {
		panic(err)
	}
}
