package memento

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/macondo/alphabet"
	"github.com/domino14/macondo/game"
)

var boardConfig = [][]rune{
	[]rune("=  '   =   '  ="),
	[]rune(" -   \"   \"   - "),
	[]rune("  -   ' '   -  "),
	[]rune("'  -   '   -  '"),
	[]rune("    -     -    "),
	[]rune(" \"   \"   \"   \" "),
	[]rune("  '   ' '   '  "),
	[]rune("=  '   *   '  ="),
	[]rune("  '   ' '   '  "),
	[]rune(" \"   \"   \"   \" "),
	[]rune("    -     -    "),
	[]rune("'  -   '   -  '"),
	[]rune("  -   ' '   -  "),
	[]rune(" -   \"   \"   - "),
	[]rune("=  '   =   '  ="),
}

// header should be pre-quantized to very few colors (ideally 8)
//go:embed header.png
var headerBytes []byte

// tiles should be pre-quantized to very few colors (ideally 48)
//go:embed tiles-english.png
var englishTilesBytes []byte

//go:embed tiles-french.png
var frenchTilesBytes []byte

//go:embed tiles-german.png
var germanTilesBytes []byte

//go:embed tiles-norwegian.png
var norwegianTilesBytes []byte

const squareDim = 68

type TilePainterTilesMeta struct {
	TilesBytes []byte
	Tile0Src   map[rune][2]int
	Tile1Src   map[rune][2]int
	BoardSrc   map[rune][2]int
	ExpDimXY   [2]int
}

var tilesMeta = map[string]*TilePainterTilesMeta{
	"english": {
		TilesBytes: englishTilesBytes,
		Tile0Src: map[rune][2]int{
			'A': {0, 0}, 'B': {68, 0}, 'C': {136, 0}, 'D': {204, 0}, 'E': {272, 0},
			'F': {340, 0}, 'G': {408, 0}, 'H': {476, 0}, 'I': {544, 0}, 'J': {612, 0},
			'K': {680, 0}, 'L': {748, 0}, 'M': {816, 0}, 'N': {884, 0}, 'O': {952, 0},
			'P': {0, 68}, 'Q': {68, 68}, 'R': {136, 68}, 'S': {204, 68}, 'T': {272, 68},
			'U': {340, 68}, 'V': {408, 68}, 'W': {476, 68}, 'X': {544, 68}, 'Y': {612, 68},
			'Z': {680, 68}, 'a': {748, 68}, 'b': {816, 68}, 'c': {884, 68}, 'd': {952, 68},
			'e': {0, 136}, 'f': {68, 136}, 'g': {136, 136}, 'h': {204, 136}, 'i': {272, 136},
			'j': {340, 136}, 'k': {408, 136}, 'l': {476, 136}, 'm': {544, 136}, 'n': {612, 136},
			'o': {680, 136}, 'p': {748, 136}, 'q': {816, 136}, 'r': {884, 136}, 's': {952, 136},
			't': {0, 204}, 'u': {68, 204}, 'v': {136, 204}, 'w': {204, 204}, 'x': {272, 204},
			'y': {340, 204}, 'z': {408, 204}, '?': {476, 204},
		},
		Tile1Src: map[rune][2]int{
			'A': {544, 204}, 'B': {612, 204},
			'C': {680, 204}, 'D': {748, 204}, 'E': {816, 204}, 'F': {884, 204}, 'G': {952, 204},
			'H': {0, 272}, 'I': {68, 272}, 'J': {136, 272}, 'K': {204, 272}, 'L': {272, 272},
			'M': {340, 272}, 'N': {408, 272}, 'O': {476, 272}, 'P': {544, 272}, 'Q': {612, 272},
			'R': {680, 272}, 'S': {748, 272}, 'T': {816, 272}, 'U': {884, 272}, 'V': {952, 272},
			'W': {0, 340}, 'X': {68, 340}, 'Y': {136, 340}, 'Z': {204, 340}, 'a': {272, 340},
			'b': {340, 340}, 'c': {408, 340}, 'd': {476, 340}, 'e': {544, 340}, 'f': {612, 340},
			'g': {680, 340}, 'h': {748, 340}, 'i': {816, 340}, 'j': {884, 340}, 'k': {952, 340},
			'l': {0, 408}, 'm': {68, 408}, 'n': {136, 408}, 'o': {204, 408}, 'p': {272, 408},
			'q': {340, 408}, 'r': {408, 408}, 's': {476, 408}, 't': {544, 408}, 'u': {612, 408},
			'v': {680, 408}, 'w': {748, 408}, 'x': {816, 408}, 'y': {884, 408}, 'z': {952, 408},
			'?': {0, 476},
		},
		BoardSrc: map[rune][2]int{
			'-': {68, 476}, '=': {136, 476}, '\'': {204, 476}, '"': {272, 476},
			'*': {340, 476}, ' ': {408, 476},
		},
		ExpDimXY: [2]int{1020, 544},
	},
	"french": {
		TilesBytes: frenchTilesBytes,
		Tile0Src: map[rune][2]int{
			'A': {0, 0}, 'B': {68, 0}, 'C': {136, 0}, 'D': {204, 0}, 'E': {272, 0},
			'F': {340, 0}, 'G': {408, 0}, 'H': {476, 0}, 'I': {544, 0}, 'J': {612, 0},
			'K': {680, 0}, 'L': {748, 0}, 'M': {816, 0}, 'N': {884, 0}, 'O': {952, 0},
			'P': {0, 68}, 'Q': {68, 68}, 'R': {136, 68}, 'S': {204, 68}, 'T': {272, 68},
			'U': {340, 68}, 'V': {408, 68}, 'W': {476, 68}, 'X': {544, 68}, 'Y': {612, 68},
			'Z': {680, 68}, 'a': {748, 68}, 'b': {816, 68}, 'c': {884, 68}, 'd': {952, 68},
			'e': {0, 136}, 'f': {68, 136}, 'g': {136, 136}, 'h': {204, 136}, 'i': {272, 136},
			'j': {340, 136}, 'k': {408, 136}, 'l': {476, 136}, 'm': {544, 136}, 'n': {612, 136},
			'o': {680, 136}, 'p': {748, 136}, 'q': {816, 136}, 'r': {884, 136}, 's': {952, 136},
			't': {0, 204}, 'u': {68, 204}, 'v': {136, 204}, 'w': {204, 204}, 'x': {272, 204},
			'y': {340, 204}, 'z': {408, 204}, '?': {476, 204},
		},
		Tile1Src: map[rune][2]int{
			'A': {544, 204}, 'B': {612, 204},
			'C': {680, 204}, 'D': {748, 204}, 'E': {816, 204}, 'F': {884, 204}, 'G': {952, 204},
			'H': {0, 272}, 'I': {68, 272}, 'J': {136, 272}, 'K': {204, 272}, 'L': {272, 272},
			'M': {340, 272}, 'N': {408, 272}, 'O': {476, 272}, 'P': {544, 272}, 'Q': {612, 272},
			'R': {680, 272}, 'S': {748, 272}, 'T': {816, 272}, 'U': {884, 272}, 'V': {952, 272},
			'W': {0, 340}, 'X': {68, 340}, 'Y': {136, 340}, 'Z': {204, 340}, 'a': {272, 340},
			'b': {340, 340}, 'c': {408, 340}, 'd': {476, 340}, 'e': {544, 340}, 'f': {612, 340},
			'g': {680, 340}, 'h': {748, 340}, 'i': {816, 340}, 'j': {884, 340}, 'k': {952, 340},
			'l': {0, 408}, 'm': {68, 408}, 'n': {136, 408}, 'o': {204, 408}, 'p': {272, 408},
			'q': {340, 408}, 'r': {408, 408}, 's': {476, 408}, 't': {544, 408}, 'u': {612, 408},
			'v': {680, 408}, 'w': {748, 408}, 'x': {816, 408}, 'y': {884, 408}, 'z': {952, 408},
			'?': {0, 476},
		},
		BoardSrc: map[rune][2]int{
			'-': {68, 476}, '=': {136, 476}, '\'': {204, 476}, '"': {272, 476},
			'*': {340, 476}, ' ': {408, 476},
		},
		ExpDimXY: [2]int{1020, 544},
	},
	"german": {
		TilesBytes: germanTilesBytes,
		Tile0Src: map[rune][2]int{
			'A': {0, 0}, 'Ä': {68, 0}, 'B': {136, 0}, 'C': {204, 0}, 'D': {272, 0},
			'E': {340, 0}, 'F': {408, 0}, 'G': {476, 0}, 'H': {544, 0}, 'I': {612, 0},
			'J': {680, 0}, 'K': {748, 0}, 'L': {816, 0}, 'M': {884, 0}, 'N': {952, 0},
			'O': {0, 68}, 'Ö': {68, 68}, 'P': {136, 68}, 'Q': {204, 68}, 'R': {272, 68},
			'S': {340, 68}, 'T': {408, 68}, 'U': {476, 68}, 'Ü': {544, 68}, 'V': {612, 68},
			'W': {680, 68}, 'X': {748, 68}, 'Y': {816, 68}, 'Z': {884, 68}, 'a': {952, 68},
			'ä': {0, 136}, 'b': {68, 136}, 'c': {136, 136}, 'd': {204, 136}, 'e': {272, 136},
			'f': {340, 136}, 'g': {408, 136}, 'h': {476, 136}, 'i': {544, 136}, 'j': {612, 136},
			'k': {680, 136}, 'l': {748, 136}, 'm': {816, 136}, 'n': {884, 136}, 'o': {952, 136},
			'ö': {0, 204}, 'p': {68, 204}, 'q': {136, 204}, 'r': {204, 204}, 's': {272, 204},
			't': {340, 204}, 'u': {408, 204}, 'ü': {476, 204}, 'v': {544, 204}, 'w': {612, 204},
			'x': {680, 204}, 'y': {748, 204}, 'z': {816, 204}, '?': {884, 204},
		},
		Tile1Src: map[rune][2]int{
			'A': {952, 204},
			'Ä': {0, 272}, 'B': {68, 272}, 'C': {136, 272}, 'D': {204, 272}, 'E': {272, 272},
			'F': {340, 272}, 'G': {408, 272}, 'H': {476, 272}, 'I': {544, 272}, 'J': {612, 272},
			'K': {680, 272}, 'L': {748, 272}, 'M': {816, 272}, 'N': {884, 272}, 'O': {952, 272},
			'Ö': {0, 340}, 'P': {68, 340}, 'Q': {136, 340}, 'R': {204, 340}, 'S': {272, 340},
			'T': {340, 340}, 'U': {408, 340}, 'Ü': {476, 340}, 'V': {544, 340}, 'W': {612, 340},
			'X': {680, 340}, 'Y': {748, 340}, 'Z': {816, 340}, 'a': {884, 340}, 'ä': {952, 340},
			'b': {0, 408}, 'c': {68, 408}, 'd': {136, 408}, 'e': {204, 408}, 'f': {272, 408},
			'g': {340, 408}, 'h': {408, 408}, 'i': {476, 408}, 'j': {544, 408}, 'k': {612, 408},
			'l': {680, 408}, 'm': {748, 408}, 'n': {816, 408}, 'o': {884, 408}, 'ö': {952, 408},
			'p': {0, 476}, 'q': {68, 476}, 'r': {136, 476}, 's': {204, 476}, 't': {272, 476},
			'u': {340, 476}, 'ü': {408, 476}, 'v': {476, 476}, 'w': {544, 476}, 'x': {612, 476},
			'y': {680, 476}, 'z': {748, 476}, '?': {816, 476},
		},
		BoardSrc: map[rune][2]int{
			'-': {884, 476}, '=': {952, 476},
			'\'': {0, 544}, '"': {68, 544}, '*': {136, 544}, ' ': {204, 544},
		},
		ExpDimXY: [2]int{1020, 612},
	},
	"norwegian": {
		TilesBytes: norwegianTilesBytes,
		Tile0Src: map[rune][2]int{
			'A': {0, 0}, 'Ä': {68, 0}, 'B': {136, 0}, 'C': {204, 0}, 'D': {272, 0},
			'E': {340, 0}, 'F': {408, 0}, 'G': {476, 0}, 'H': {544, 0}, 'I': {612, 0},
			'J': {680, 0}, 'K': {748, 0}, 'L': {816, 0}, 'M': {884, 0}, 'N': {952, 0},
			'O': {0, 68}, 'Ö': {68, 68}, 'P': {136, 68}, 'Q': {204, 68}, 'R': {272, 68},
			'S': {340, 68}, 'T': {408, 68}, 'U': {476, 68}, 'Ü': {544, 68}, 'V': {612, 68},
			'W': {680, 68}, 'X': {748, 68}, 'Y': {816, 68}, 'Z': {884, 68}, 'Æ': {952, 68},
			'Ø': {0, 136}, 'Å': {68, 136}, 'a': {136, 136}, 'ä': {204, 136}, 'b': {272, 136},
			'c': {340, 136}, 'd': {408, 136}, 'e': {476, 136}, 'f': {544, 136}, 'g': {612, 136},
			'h': {680, 136}, 'i': {748, 136}, 'j': {816, 136}, 'k': {884, 136}, 'l': {952, 136},
			'm': {0, 204}, 'n': {68, 204}, 'o': {136, 204}, 'ö': {204, 204}, 'p': {272, 204},
			'q': {340, 204}, 'r': {408, 204}, 's': {476, 204}, 't': {544, 204}, 'u': {612, 204},
			'ü': {680, 204}, 'v': {748, 204}, 'w': {816, 204}, 'x': {884, 204}, 'y': {952, 204},
			'z': {0, 272}, 'æ': {68, 272}, 'ø': {136, 272}, 'å': {204, 272}, '?': {272, 272},
		},
		Tile1Src: map[rune][2]int{
			'A': {340, 272}, 'Ä': {408, 272}, 'B': {476, 272}, 'C': {544, 272}, 'D': {612, 272},
			'E': {680, 272}, 'F': {748, 272}, 'G': {816, 272}, 'H': {884, 272}, 'I': {952, 272},
			'J': {0, 340}, 'K': {68, 340}, 'L': {136, 340}, 'M': {204, 340}, 'N': {272, 340},
			'O': {340, 340}, 'Ö': {408, 340}, 'P': {476, 340}, 'Q': {544, 340}, 'R': {612, 340},
			'S': {680, 340}, 'T': {748, 340}, 'U': {816, 340}, 'Ü': {884, 340}, 'V': {952, 340},
			'W': {0, 408}, 'X': {68, 408}, 'Y': {136, 408}, 'Z': {204, 408}, 'Æ': {272, 408},
			'Ø': {340, 408}, 'Å': {408, 408}, 'a': {476, 408}, 'ä': {544, 408}, 'b': {612, 408},
			'c': {680, 408}, 'd': {748, 408}, 'e': {816, 408}, 'f': {884, 408}, 'g': {952, 408},
			'h': {0, 476}, 'i': {68, 476}, 'j': {136, 476}, 'k': {204, 476}, 'l': {272, 476},
			'm': {340, 476}, 'n': {408, 476}, 'o': {476, 476}, 'ö': {544, 476}, 'p': {612, 476},
			'q': {680, 476}, 'r': {748, 476}, 's': {816, 476}, 't': {884, 476}, 'u': {952, 476},
			'ü': {0, 544}, 'v': {68, 544}, 'w': {136, 544}, 'x': {204, 544}, 'y': {272, 544},
			'z': {340, 544}, 'æ': {408, 544}, 'ø': {476, 544}, 'å': {544, 544}, '?': {612, 544},
		},
		BoardSrc: map[rune][2]int{
			'-': {680, 544}, '=': {748, 544}, '\'': {816, 544}, '"': {884, 544}, '*': {952, 544},
			' ': {0, 612},
		},
		ExpDimXY: [2]int{1020, 680},
	},
}

type LoadedTilesImg struct {
	tilesImg image.Image
	palette  []color.Color
}

func loadTilesImg(tptm *TilePainterTilesMeta, headerPal map[color.Color]struct{}) (*LoadedTilesImg, error) {
	img, err := png.Decode(bytes.NewReader(tptm.TilesBytes))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	expectedX := tptm.ExpDimXY[0]
	expectedY := tptm.ExpDimXY[1]
	if bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Dx() != expectedX || bounds.Dy() != expectedY {
		return nil, fmt.Errorf("unexpected size: %s vs %s", bounds.String(), image.Pt(expectedX, expectedY))
	}

	// Build an up to 256 colors palette where index 0 is Transparent.
	inPal := make(map[color.Color]struct{})
	for k := range headerPal {
		inPal[k] = struct{}{}
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			inPal[img.At(x, y)] = struct{}{}
		}
	}
	pal := make([]color.Color, 0, len(inPal)+1)
	// Always put image.Transparent even if there is another color with zero alpha.
	pal = append(pal, image.Transparent)
	for k := range inPal {
		pal = append(pal, k)
	}
	if len(pal) > 256 {
		return nil, fmt.Errorf("gif cannot support %d colors", len(pal))
	}
	// Sort deterministically, exclude the image.Transparent.
	sort.Slice(pal[1:], func(i, j int) bool {
		ri, gi, bi, ai := pal[i+1].RGBA()
		rj, gj, bj, aj := pal[j+1].RGBA()
		if ai != aj {
			return ai < aj
		}
		if ri != rj {
			return ri < rj
		}
		if gi != gj {
			return gi < gj
		}
		return bi < bj
	})

	return &LoadedTilesImg{
		tilesImg: img,
		palette:  pal,
	}, nil
}

var tilesImgCache map[string]*LoadedTilesImg

// using *image.NRGBA directly instead of image.Image might be slightly faster?
type prerenderedBackgroundsType struct {
	standardBoard map[string]*image.NRGBA
}

var prerenderedBackgroundsCache prerenderedBackgroundsType

var padTop, padRight, padBottom, padLeft = 10, 10, 10, 10
var padHeader = 10
var headerHeight, ofsTop int
var paddingColor color.Color

func init() {
	headerImg, err := png.Decode(bytes.NewReader(headerBytes))
	if err != nil {
		panic(fmt.Errorf("can't load header png: %v", err))
	}
	headerPal := make(map[color.Color]struct{})
	bounds := headerImg.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			headerPal[headerImg.At(x, y)] = struct{}{}
		}
	}
	headerHeight = bounds.Dy()
	paddingColor = headerImg.At(0, 0) // use top left pixel color
	ofsTop = padTop + headerHeight + padHeader + 1

	backgroundImgs := make(map[string]*image.NRGBA)

	nRows := len(boardConfig)
	if nRows < 1 {
		panic(fmt.Errorf("invalid boardConfig: expecting at least 1 row"))
	}

	nCols := len(boardConfig[0])
	for i, row := range boardConfig {
		if i > 0 && len(row) != nCols {
			panic(fmt.Errorf("invalid boardConfig: expecting row %d to have length %d", i+1, nCols))
		}
	}

	ret := make(map[string]*LoadedTilesImg)
	for k, tptm := range tilesMeta {
		loadedTilesImg, err := loadTilesImg(tptm, headerPal)
		if err != nil {
			panic(fmt.Errorf("can't load tilesImg for %s: %v", k, err))
		}
		ret[k] = loadedTilesImg

		backgroundImg := image.NewNRGBA(image.Rect(0, 0, padLeft+nCols*squareDim+1+padRight, ofsTop+nRows*squareDim+padBottom))
		draw.Draw(backgroundImg, backgroundImg.Bounds(), &image.Uniform{paddingColor}, image.ZP, draw.Src)
		headerImgRight := padLeft + headerImg.Bounds().Dx()
		headerImgRightCannotExceed := backgroundImg.Bounds().Dx() - padRight
		if headerImgRightCannotExceed < headerImgRight {
			headerImgRight = headerImgRightCannotExceed
		}
		draw.Draw(backgroundImg, image.Rect(padLeft, padTop, headerImgRight, padTop+headerHeight), headerImg, image.ZP, draw.Over)
		for r := 0; r < nRows; r++ {
			for c := 0; c < nCols; c++ {
				drawEmptySquare(tptm, loadedTilesImg.tilesImg, backgroundImg, r, c, boardConfig[r][c])
			}
		}

		// Missing borders. Add 1 px at top and right.
		srcPt := tptm.BoardSrc[' '] // has bottom and left borders
		srcl := srcPt[0]
		srct := srcPt[1]
		srcb := srct + squareDim - 1
		// Copy bottom border to top of board.
		// This must be what aboveboard means.
		y := ofsTop - 1
		x := padLeft
		for c := 0; c < nCols; c++ {
			draw.Draw(backgroundImg, image.Rect(x, y, x+squareDim, y+1), loadedTilesImg.tilesImg,
				image.Pt(srcl, srcb), draw.Src)
			x += squareDim
		}
		// Copy bottom-left pixel of sample to top right of board.
		draw.Draw(backgroundImg, image.Rect(x, y, x+1, y+1), loadedTilesImg.tilesImg,
			image.Pt(srcl, srcb), draw.Src)
		y += 1
		// Copy left border to right.
		for r := 0; r < nRows; r++ {
			draw.Draw(backgroundImg, image.Rect(x, y, x+1, y+squareDim), loadedTilesImg.tilesImg,
				image.Pt(srcl, srct), draw.Src)
			y += squareDim
		}

		backgroundImgs[k] = backgroundImg
	}

	tilesImgCache = ret
	prerenderedBackgroundsCache = prerenderedBackgroundsType{
		standardBoard: backgroundImgs,
	}
}

func drawEmptySquare(tptm *TilePainterTilesMeta, tilesImg image.Image, img *image.NRGBA, r, c int, b rune) {
	y := r*squareDim + ofsTop
	x := c*squareDim + padLeft
	srcPt, ok := tptm.BoardSrc[rune(b)]
	if !ok {
		srcPt = tptm.BoardSrc[' ']
	}
	draw.Draw(img, image.Rect(x, y, x+squareDim, y+squareDim), tilesImg,
		image.Pt(srcPt[0], srcPt[1]), draw.Src)
}

func realWhose(whichColor int, actualWhose byte) byte {
	switch whichColor {
	case 0:
		return 0
	case 1:
		return 1
	default:
		return actualWhose
	}
}

func drawTileOnBoard(tptm *TilePainterTilesMeta, tilesImg image.Image, img *image.NRGBA, r, c int, b rune, p byte) {
	y := r*squareDim + ofsTop
	x := c*squareDim + padLeft
	if b != ' ' {
		tSrc := tptm.Tile0Src
		if p&1 != 0 {
			tSrc = tptm.Tile1Src
		}
		srcPt, ok := tSrc[b]
		if !ok {
			srcPt = tSrc['?']
		}
		draw.Draw(img, image.Rect(x, y, x+squareDim, y+squareDim), tilesImg,
			image.Pt(srcPt[0], srcPt[1]), draw.Over)
	}
}

func whichFromEvent(history *macondopb.GameHistory, evt *macondopb.GameEvent) byte {
	which := byte(0)
	if len(history.Players) >= 2 {
		if evt.Nickname != history.Players[0].Nickname {
			which = 1
		}
		if history.SecondWentFirst {
			which ^= 1 // Fix coloring. WHY.
		}
	}
	return which
}

func RenderImage(history *macondopb.GameHistory, wf WhichFile) ([]byte, error) {
	isStatic := wf.FileType == "png"
	numEvents := math.MaxInt
	if wf.HasNextEventNum {
		numEvents = wf.NextEventNum - 1
	}

	_, letterDistributionName, _ := game.HistoryToVariant(history)
	lang := strings.TrimSuffix(letterDistributionName, "_super")

	tptm, ok := tilesMeta[lang]
	if !ok {
		return nil, fmt.Errorf("missing tilesMeta: " + lang)
	}

	loadedTilesImg, ok := tilesImgCache[lang]
	if !ok {
		return nil, fmt.Errorf("missing tilesImgCache: " + lang)
	}
	tilesImg := loadedTilesImg.tilesImg
	palette := loadedTilesImg.palette

	prerenderedBackgroundCache := prerenderedBackgroundsCache.standardBoard
	backgroundImg, ok := prerenderedBackgroundCache[lang]
	if !ok {
		return nil, fmt.Errorf("missing prerenderedBackgroundCache: " + lang)
	}
	backgroundImgBounds := backgroundImg.Bounds()
	singleImg := image.NewNRGBA(backgroundImgBounds)
	draw.Draw(singleImg, backgroundImgBounds, backgroundImg, image.ZP, draw.Src)

	nRows := len(boardConfig)
	nCols := len(boardConfig[0])
	onBoard := func(r, c int) bool {
		return r >= 0 && r < nRows && c >= 0 && c < nCols
	}

	agif := &gif.GIF{}
	addFrame := func(img *image.NRGBA, delay int, op draw.Op) {
		imgPal := image.NewPaletted(img.Bounds(), palette)
		draw.Draw(imgPal, imgPal.Bounds(), img, img.Bounds().Min, op)
		agif.Image = append(agif.Image, imgPal)
		agif.Delay = append(agif.Delay, delay)
	}
	if isStatic {
		addFrame = func(img *image.NRGBA, delay int, op draw.Op) {}
	}
	addFrame(singleImg, 50, draw.Src)

	makeSubImage := func(rect image.Rectangle) *image.NRGBA {
		return image.NewNRGBA(rect)
	}
	if isStatic {
		makeSubImage = func(rect image.Rectangle) *image.NRGBA {
			return singleImg
		}
	}

	patchImage := func(evt *macondopb.GameEvent, callback func(img *image.NRGBA, r, c int, ch rune)) {
		r, c := int(evt.Row), int(evt.Column)
		dr, dc := 0, 1
		if evt.Direction == macondopb.GameEvent_VERTICAL {
			dr, dc = 1, 0
		}
		str := evt.PlayedTiles
		for {
			ru, size := utf8.DecodeRuneInString(str)
			if ru != alphabet.ASCIIPlayedThrough {
				break
			}
			r, c = r+dr, c+dc
			str = str[size:]
		}
		if len(str) == 0 {
			return
		}
		for {
			ru, size := utf8.DecodeLastRuneInString(str)
			if ru != alphabet.ASCIIPlayedThrough {
				break
			}
			str = str[:len(str)-size]
		}
		numPlayedTiles := utf8.RuneCountInString(str)
		img := makeSubImage(image.Rect(c*squareDim+padLeft, r*squareDim+ofsTop, (c+1+(numPlayedTiles-1)*dc)*squareDim+padLeft, (r+1+(numPlayedTiles-1)*dr)*squareDim+ofsTop))
		for _, ch := range str {
			if ch != alphabet.ASCIIPlayedThrough {
				callback(img, r, c, ch)
			}
			r, c = r+dr, c+dc
		}
		addFrame(img, 50, draw.Over)
	}
	lastPlaceIndex := -1
	for i, evt := range history.Events {
		if i >= numEvents {
			break
		}
		switch evt.GetType() {
		case macondopb.GameEvent_TILE_PLACEMENT_MOVE:
			lastPlaceIndex = i
			which := whichFromEvent(history, evt)
			patchImage(evt, func(img *image.NRGBA, r, c int, ch rune) {
				if onBoard(r, c) {
					drawTileOnBoard(tptm, tilesImg, img, r, c, ch, realWhose(wf.WhichColor, which))
				}
			})
		case macondopb.GameEvent_PHONY_TILES_RETURNED:
			if lastPlaceIndex >= 0 {
				patchImage(history.Events[lastPlaceIndex], func(img *image.NRGBA, r, c int, ch rune) {
					if onBoard(r, c) {
						drawEmptySquare(tptm, tilesImg, img, r, c, boardConfig[r][c])
					}
				})
				lastPlaceIndex = -1
			}
		}
	}

	// We want the final frame to stay for 2 sec.
	// Chrome interprets Delay as the delay after the frame.
	// Mac Quick Look interprets Delay as the delay before the frame.
	// So if we set the last frame's delay to 200cs (for Chrome),
	// Mac Quick Look delays the next-to-last frame instead.
	// If we set the first frame's delay to 200cs (for Mac Quick Look),
	// Chrome delays the first frame instead.
	// Solution: we add a transparent 1x1 frame and run it for 150cs.
	// This adds about 215 bytes to the file, but works for both.
	addFrame(image.NewNRGBA(image.Rect(0, 0, 1, 1)), 150, draw.Over)

	var buf bytes.Buffer
	var err error
	if isStatic {
		err = png.Encode(&buf, singleImg)
	} else {
		err = gif.EncodeAll(&buf, agif)
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
