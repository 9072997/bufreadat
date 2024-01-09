package bufreadat

import (
	"bytes"
	"crypto/rand"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestReaderAt_rangeToBlockRange(t *testing.T) {
	ra := &ReaderAt{blockSize: 100}
	testCases := [][2][2]int64{
		{{0, 0}, {0, 0}}, // empty range
		{{0, 1}, {0, 1}},
		{{0, 2}, {0, 1}},
		{{0, 98}, {0, 1}},
		{{0, 99}, {0, 1}},
		{{0, 100}, {0, 1}},
		{{0, 101}, {0, 2}},
		{{0, 102}, {0, 2}},
		{{0, 198}, {0, 2}},
		{{0, 199}, {0, 2}},
		{{0, 200}, {0, 2}},
		{{0, 201}, {0, 3}},
		{{0, 202}, {0, 3}},

		{{1, 1}, {0, 0}}, // empty range
		{{1, 2}, {0, 1}},
		{{1, 98}, {0, 1}},
		{{1, 99}, {0, 1}},
		{{1, 100}, {0, 1}},
		{{1, 101}, {0, 2}},
		{{1, 102}, {0, 2}},
		{{1, 198}, {0, 2}},
		{{1, 199}, {0, 2}},
		{{1, 200}, {0, 2}},
		{{1, 201}, {0, 3}},
		{{1, 202}, {0, 3}},

		{{98, 98}, {0, 0}}, // empty range
		{{98, 99}, {0, 1}},
		{{98, 100}, {0, 1}},
		{{98, 101}, {0, 2}},
		{{98, 102}, {0, 2}},

		{{99, 99}, {0, 0}}, // empty range
		{{99, 100}, {0, 1}},
		{{99, 101}, {0, 2}},
		{{99, 102}, {0, 2}},

		{{100, 100}, {1, 1}}, // empty range
		{{100, 101}, {1, 2}},
		{{100, 102}, {1, 2}},

		{{101, 101}, {1, 1}}, // empty range
		{{101, 102}, {1, 2}},
	}
	for _, tc := range testCases {
		start, end := ra.rangeToBlockRange(tc[0][0], tc[0][1])
		if start != tc[1][0] || end != tc[1][1] {
			t.Fatalf(
				"rangeToBlockRange(%d, %d) = (%d, %d), want (%d, %d)",
				tc[0][0], tc[0][1],
				start, end,
				tc[1][0], tc[1][1],
			)
		}
	}
}

func TestReaderAt_blockRangeToRange(t *testing.T) {
	ra := &ReaderAt{blockSize: 100}
	testCases := [][2][2]int64{
		{{0, 0}, {0, 0}}, // empty range
		{{0, 1}, {0, 100}},
		{{0, 2}, {0, 200}},

		{{1, 1}, {100, 100}}, // empty range
		{{1, 2}, {100, 200}},

		{{2, 2}, {200, 200}}, // empty range
		{{2, 3}, {200, 300}},
	}
	for _, tc := range testCases {
		start, end := ra.blockRangeToRange(tc[0][0], tc[0][1])
		if start != tc[1][0] || end != tc[1][1] {
			t.Fatalf(
				"blockRangeToRange(%d, %d) = (%d, %d), want (%d, %d)",
				tc[0][0], tc[0][1],
				start, end,
				tc[1][0], tc[1][1],
			)
		}
	}
}

const TestCacheSize = 2
const TestBlockSize = 4

func getBufReaderAt() *ReaderAt {
	r := bytes.NewReader([]byte(strings.Repeat("0123456789abcdefghijklmnopqrstuvwxyz", 128*4)))
	return New(r, TestBlockSize, TestCacheSize)
}

func getCustomeBufReaderAt(size int, toRead string) *ReaderAt {
	r := bytes.NewReader([]byte(toRead))
	return New(r, int64(size), TestCacheSize)
}

func TestReadAt0(t *testing.T) {
	r := getBufReaderAt()

	bufSize := 2

	b := make([]byte, bufSize)
	n, err := r.ReadAt(b, 0)
	if err != nil {
		t.Fatal(err)
	}
	if n != bufSize {
		t.Errorf("n didn't match: %d", n)
	}
	if string(b) != "01" {
		t.Errorf("read result didn't match: %v", b)
	}
}

func TestReadAt1(t *testing.T) {
	r := getBufReaderAt()

	bufSize := 3

	b := make([]byte, bufSize)
	n, err := r.ReadAt(b, 4)
	if err != nil {
		t.Fatal(err)
	}
	if n != bufSize {
		t.Fatal("n didn't match:", n)
	}

	n, err = r.ReadAt(b, 1)
	if err != nil {
		t.Fatal(err)
	}
	if n != bufSize {
		t.Error("n didn't match:", n)
	}
	if n != bufSize {
		t.Error("n didn't match:", n)
	}
	if string(b) != "123" {
		t.Errorf("read result didn't match: %v", b)
	}
}

func TestReadAt2(t *testing.T) {
	r := getBufReaderAt()

	bufSize := 3
	var offset int64 = 3

	b := make([]byte, bufSize)
	n, err := r.ReadAt(b, offset)
	if err != nil {
		t.Fatal(err)
	}
	if n != bufSize {
		t.Errorf("n didn't match: %d", n)
	}
	if string(b) != "345" {
		t.Errorf("read result didn't match: %v", b)
	}
}

func TestReadAt3(t *testing.T) {
	r := getBufReaderAt()

	bufSize := 3
	b := make([]byte, bufSize)

	// set cache
	n, err := r.ReadAt(b, 9)
	if err != nil {
		t.Fatalf("err wasn't nil: %+v", err)
	}
	if n != bufSize {
		t.Fatalf("n didn't match: %d", n)
	}

	// read from cache
	n, err = r.ReadAt(b, 9)
	if err != nil {
		t.Fatal("err wasn't nil")
	}
	if n != bufSize {
		t.Error("n didn't match:", n)
	}
	if string(b) != "9ab" {
		t.Errorf("read result didn't match: %v", b)
	}

}

func TestReadAt4(t *testing.T) {
	r := getBufReaderAt()

	bufSize := 3
	var offset int64 = 6

	b := make([]byte, bufSize)
	n, err := r.ReadAt(b, offset)
	if err != nil {
		t.Fatal(err)
	}
	if n != bufSize {
		t.Errorf("n didn't match: %d", n)
	}
	if string(b) != "678" {
		t.Errorf("read result didn't match: %v", b)
	}
}

func TestReadAt5(t *testing.T) {
	r := getBufReaderAt()

	bufSize := 3

	b := make([]byte, bufSize)
	n, err := r.ReadAt(b, 4)
	if err != nil {
		t.Fatal(err)
	}
	if n != bufSize {
		t.Fatal("n didn't match:", n)
	}

	n, err = r.ReadAt(b, 9)
	if err != nil {
		t.Fatal(err)
	}
	if n != bufSize {
		t.Error("n didn't match:", n)
	}
	if n != bufSize {
		t.Error("n didn't match:", n)
	}
	if string(b) != "9ab" {
		t.Errorf("read result didn't match: %v", b)
	}
}

func TestReadAt6(t *testing.T) {
	r := getCustomeBufReaderAt(4, "01234567")

	bufSize := 3
	b := make([]byte, bufSize)

	n, err := r.ReadAt(b, 5)
	if err != nil {
		t.Errorf("err wasn't nil: %+v", err)
	}
	if n != bufSize {
		t.Errorf("n didn't match: %d", n)
	}
	if string(b) != "567" {
		t.Errorf("read result didn't match: %v", b)
	}

	n, err = r.ReadAt(b, 8)
	if err != io.EOF {
		t.Error("err wasn't io.EOF:", err)
	}
	if n != 0 {
		t.Errorf("n isn't 0: %d", n)
	}
}

func TestReadAt7(t *testing.T) {
	r := getCustomeBufReaderAt(4, "012345")

	bufSize := 3
	b := make([]byte, bufSize)

	n, err := r.ReadAt(b, 4)

	if err != io.EOF {
		t.Errorf("err wasn't io.EOF: %+v", err)
	}
	if n != 2 {
		t.Errorf("n isn't 0: %d", n)
	}
	if string(b[:n]) != "45" {
		t.Errorf("read result didn't match: %v", b)
	}
}

func TestReadAt8(t *testing.T) {
	r := getCustomeBufReaderAt(4, "0123456")

	bufSize := 2
	b := make([]byte, bufSize)

	// renew
	n, err := r.ReadAt(b, 4)
	if err != nil {
		t.Error("err wasn't nil:", err)
	}
	if n != 2 {
		t.Error("n isn't 0:", n)
	}
	if string(b[:n]) != "45" {
		t.Errorf("read result didn't match: %v", b)
	}

	// just read
	n, err = r.ReadAt(b, 4)
	if err != nil {
		t.Error("err wasn't nil:", err)
	}
	if n != 2 {
		t.Error("n isn't 0:", n)
	}
	if string(b[:n]) != "45" {
		t.Errorf("read result didn't match: %v", b)
	}

}
func TestReadAt9(t *testing.T) {
	r := getCustomeBufReaderAt(4, "0123456")

	bufSize := 2
	b := make([]byte, bufSize)

	// renew
	n, err := r.ReadAt(b, 6)
	if err != io.EOF {
		t.Error("err wasn't io.EOF:", err)
	}
	if n != 1 {
		t.Error("n isn't 0:", n)
	}
	if string(b[:n]) != "6" {
		t.Errorf("read result didn't match: %v", b)
	}

	// just read
	n, err = r.ReadAt(b, 6)
	if err != io.EOF {
		t.Error("err wasn't io.EOF:", err)
	}
	if n != 1 {
		t.Error("n isn't 0:", n)
	}
	if string(b[:n]) != "6" {
		t.Errorf("read result didn't match: %v", b)
	}
}

func TestReadAt10(t *testing.T) {
	orig := strings.Repeat("0123456789abcdefghijklmnopqrstuvwxyz", 128*4)
	r := getCustomeBufReaderAt(17, orig)
	bufSize := 11
	b := make([]byte, bufSize)

	result := &bytes.Buffer{}
	offset := int64(0)

	for {
		n, err := r.ReadAt(b, offset)

		offset += int64(n)

		result.Write(b[:n])
		if err != nil {
			break
		}

	}

	if result.String() != orig {
		t.Error("result didn't match")
	}

}

// io.ReaderAt specifies clients of ReadAt can execute parallel ReadAt calls
// on the same input source
func TestConcurrency(t *testing.T) {
	// fill a buffer with random data
	buf := make([]byte, 16*1024) // 16KB
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		t.Fatal(err)
	}

	// create a reader for the buffer and a BufReaderAt for the reader
	r := bytes.NewReader([]byte(buf))
	buffered := New(r, 128, 2)

	// start a bunch of threads
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		go func(i int) {
			for j := 0; j < 1000; j++ {
				// pick a random position and length
				iterator := 1000*i + j // 0 - 99,999
				startPos := iterator % (len(buf) - 1000)
				length := j%7919 + 1
				endPos := startPos + length

				// read the data directly from the buffer and from the BufReaderAt
				expected := buf[startPos:endPos]
				actual := make([]byte, length)
				_, err := buffered.ReadAt(actual, int64(startPos))
				if err != nil {
					t.Log(err)
					t.Fail()
				}

				// compare the results
				if !bytes.Equal(expected, actual) {
					t.Logf("expected %v, got %v", expected, actual)
					t.Fail()
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestEfficiency(t *testing.T) {
	// fill a buffer with random data
	buf := make([]byte, 16*1024) // 16KB
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		t.Fatal(err)
	}

	// create a reader for the buffer and a BufReaderAt for the reader
	r := bytes.NewReader([]byte(buf))
	buffered := New(r, 100, 82) // ~8KB (1/2 total buffer size)

	for pass := 0; pass < 2; pass++ {
		// perform a bunch of 1k reads, slowly increasing the offset
		for i := 0; i < 15*1024; i++ {
			expected := buf[i : i+1024]
			actual := make([]byte, 1024)
			_, err := buffered.ReadAt(actual, int64(i))
			if err != nil {
				t.Fatal(err)
			}

			// compare the results
			if !bytes.Equal(expected, actual) {
				t.Fatalf("expected %v, got %v", expected, actual)
			}
		}
	}

	// check the stats
	overBytes, underBytes, overReqs, underReqs := buffered.Stats()
	byteImprovement := float64(overBytes) / float64(underBytes)
	reqImprovement := float64(overReqs) / float64(underReqs)
	t.Logf(
		"overBytes: %d, underBytes: %d, improvement: %.2fx",
		overBytes, underBytes, byteImprovement,
	)
	t.Logf(
		"overReqs: %d, underReqs: %d, improvement: %.2fx",
		overReqs, underReqs, reqImprovement,
	)
	if underBytes != 32*1024 {
		t.Fatalf(
			"underBytes should be 32KB (2x 16KB passes), got %d",
			underBytes,
		)
	}
	if byteImprovement < 900 {
		t.Fatalf(
			"expected ~1000x byte improvement, got %.2fx",
			byteImprovement,
		)
	}
	if reqImprovement < 90 {
		t.Fatalf(
			"expected ~100x req improvement, got %.2fx",
			reqImprovement,
		)
	}
}
