package bufreadat

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

const emptyBrailleCell = rune(0x2800)
const dotMask1 = rune(0x0001) // [⠁]
const dotMask2 = rune(0x0002) // [⠂]
const dotMask3 = rune(0x0004) // [⠄]
const dotMask4 = rune(0x0008) // [⠈]
const dotMask5 = rune(0x0010) // [⠐]
const dotMask6 = rune(0x0020) // [⠠]
const dotMask7 = rune(0x0040) // [⡀]
const dotMask8 = rune(0x0080) // [⢀]

func (r *ReaderAt) brailleLine() string {
	lineLen := utf8.RuneCountInString(r.prevLine)
	line := make([]rune, lineLen)
	for i := 0; i < lineLen; i++ {
		line[i] = emptyBrailleCell
	}

	blockCount := r.fileLen / r.blockSize
	if r.fileLen%blockCount != 0 {
		blockCount++
	}
	for i := range r.cache {
		position := (2 * int64(lineLen) * i) / blockCount
		positionNext := (2 * int64(lineLen) * (i + 1)) / blockCount
		for j := position; j < positionNext; j++ {
			charPos := j / 2
			c := &line[charPos]
			if j%2 == 0 {
				*c |= dotMask1 // left dot
			} else {
				*c |= dotMask4 // right dot
			}
		}
	}

	return string(line)
}

// returns 0-3 indicating the first empty row of the braille line
// or 4 if the line is full
func firstEmptyRow(line string) int {
	var rows [4]bool
	for _, c := range line {
		if c&dotMask1 != 0 || c&dotMask4 != 0 {
			rows[0] = true
		}
		if c&dotMask2 != 0 || c&dotMask5 != 0 {
			rows[1] = true
		}
		if c&dotMask3 != 0 || c&dotMask6 != 0 {
			rows[2] = true
		}
		if c&dotMask7 != 0 || c&dotMask8 != 0 {
			rows[3] = true
		}
	}
	for i, row := range rows {
		if !row {
			return i
		}
	}
	return 4
}

func mergeBraille(prev, next string) string {
	var prevRunes []rune
	for _, c := range prev {
		prevRunes = append(prevRunes, c)
	}
	var nextRunes []rune
	for _, c := range next {
		nextRunes = append(nextRunes, c)
	}

	if len(prevRunes) != len(nextRunes) {
		panic("mismatched braille line lengths")
	}
	row := firstEmptyRow(prev)

	var merged []rune
	for i, p := range prevRunes {
		leftCol := nextRunes[i]&dotMask1 != 0
		rightCol := nextRunes[i]&dotMask4 != 0
		switch row {
		case 0:
			panic("empty row")
		case 1:
			if leftCol {
				p |= dotMask2
			}
			if rightCol {
				p |= dotMask5
			}
		case 2:
			if leftCol {
				p |= dotMask3
			}
			if rightCol {
				p |= dotMask6
			}
		case 3:
			if leftCol {
				p |= dotMask7
			}
			if rightCol {
				p |= dotMask8
			}
		default:
			panic("full row")
		}
		merged = append(merged, p)
	}

	return string(merged)
}

func (r *ReaderAt) drawGraph() {
	if r.prevLine == "" {
		return
	}
	subLine := r.brailleLine()
	switch firstEmptyRow(r.prevLine) {
	case 0:
		r.prevLine = subLine
		fmt.Print(r.prevLine)
	case 1, 2, 3:
		r.prevLine = mergeBraille(r.prevLine, subLine)
		fmt.Printf("\r%s", r.prevLine)
	case 4:
		r.prevLine = subLine
		fmt.Printf("\n%s", r.prevLine)
	default:
		panic("invalid row")
	}
}

// After calling this method, the ReaderAt will draw a graph of the cache
// to to stdout every time a read request is made to the underlying reader.
func (r *ReaderAt) EnableGraph(fileLen int64) error {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	if termWidth <= 0 {
		return fmt.Errorf("invalid terminal width: %d", termWidth)
	}
	for i := 0; i < termWidth; i++ {
		fmt.Print("=")
	}
	fmt.Println()
	r.prevLine = strings.Repeat(string(emptyBrailleCell), termWidth)
	r.fileLen = fileLen
	runtime.SetFinalizer(&r.cache, func(_ map[int64]*cacheEntry) {
		fmt.Println()
	})
	return nil
}
