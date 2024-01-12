package bufreadat

import (
	"io"
	"sync"
	"sync/atomic"
)

type cacheEntry struct {
	lastAccessed atomic.Uint64
	data         []byte
}

type ReaderAt struct {
	underlying io.ReaderAt
	blockSize  int64
	numBlocks  int64
	cache      map[int64]*cacheEntry
	cacheMutex sync.RWMutex
	clock      atomic.Uint64
	// for stats
	overlayBytes  atomic.Uint64
	underlayBytes uint64
	overlayReqs   atomic.Uint64
	underlayReqs  uint64
	// for graph
	fileLen  int64
	prevLine string
}

type readRequest struct {
	buffer []byte
	offset int64
	err    error
}

func New(
	underlying io.ReaderAt, blockSize int64, numBlocks int64,
) *ReaderAt {
	return &ReaderAt{
		underlying: underlying,
		blockSize:  blockSize,
		numBlocks:  numBlocks,
		cache:      make(map[int64]*cacheEntry),
	}
}

func (r *ReaderAt) canBeServedFromCache(rr *readRequest) bool {
	// can we read the whole request from the cache?
	startBlock, endBlock := r.rangeToBlockRange(
		rr.offset,
		rr.offset+int64(len(rr.buffer)),
	)
	for i := startBlock; i < endBlock; i++ {
		_, inCache := r.cache[i]
		if !inCache {
			return false
		}
	}
	return true
}

func (r *ReaderAt) processReadRequest(rr *readRequest) {
	// get block range
	startBlock, endBlock := r.rangeToBlockRange(
		rr.offset,
		rr.offset+int64(len(rr.buffer)),
	)
	// how many new blocks do we need to read?
	var numNewBlocks int64
	for i := startBlock; i < endBlock; i++ {
		_, inCache := r.cache[i]
		if !inCache {
			numNewBlocks++
		}
	}
	// how many blocks do we need to evict?
	numOldBlocks := int64(len(r.cache))
	numToEvict := numOldBlocks + numNewBlocks - r.numBlocks
	// evict blocks
	for numToEvict > 0 {
		var oldestBlock int64
		oldestTime := r.clock.Load()
		for block, entry := range r.cache {
			lastAccessed := entry.lastAccessed.Load()
			if lastAccessed < oldestTime {
				oldestBlock = block
				oldestTime = lastAccessed
			}
		}
		delete(r.cache, oldestBlock)
		numToEvict--
	}

	// group missing blocks into ranges
	var missingRanges [][2]int64
	prevInCache := true
	for i := startBlock; i < endBlock; i++ {
		_, inCache := r.cache[i]
		if inCache == prevInCache {
			// not an edge
			continue
		}
		if inCache {
			// end of a missing range
			current := len(missingRanges) - 1
			missingRanges[current][1] = i
		} else {
			// start of a missing range
			missingRanges = append(missingRanges, [2]int64{i, i})
		}
		prevInCache = inCache
	}
	if !prevInCache {
		// end final range
		current := len(missingRanges) - 1
		missingRanges[current][1] = endBlock
	}
	// read missing ranges
	for _, missingRange := range missingRanges {
		mrStartBlock := missingRange[0]
		mrEndBlock := missingRange[1]
		mrStart, mrEnd := r.blockRangeToRange(mrStartBlock, mrEndBlock)
		mrLen := mrEnd - mrStart
		mrBuffer := make([]byte, mrLen)
		n, err := r.underlying.ReadAt(mrBuffer, mrStart)
		r.underlayBytes += uint64(n)
		r.underlayReqs++
		if err != nil && err != io.EOF {
			rr.buffer = nil
			rr.err = err
			return
		}
		mrBuffer = mrBuffer[:n]
		mbBufferPtr := int64(0)
		// save blocks to cache
		for i := mrStartBlock; i < mrEndBlock; i++ {
			blockLen := r.blockSize
			remaining := int64(len(mrBuffer)) - mbBufferPtr
			if remaining < blockLen {
				blockLen = remaining
			}
			blockBuffer := mrBuffer[mbBufferPtr : mbBufferPtr+blockLen]
			r.cache[i] = &cacheEntry{
				data: blockBuffer,
			}
			mbBufferPtr += blockLen
		}
	}

	// calculate how much "waste" there is from the first block
	// due to un-aligned reads
	brStart, _ := r.blockRangeToRange(startBlock, endBlock)
	waste := rr.offset - brStart
	// copy data from cache to buffer
	bufferPtr := 0
	for i := startBlock; i < endBlock; i++ {
		entry := r.cache[i]
		entry.lastAccessed.Store(r.clock.Add(1))
		bufferPtr += copy(rr.buffer[bufferPtr:], entry.data[waste:])
		waste = 0
	}
	// truncate buffer if read overran end of file)
	if len(rr.buffer) > bufferPtr {
		rr.buffer = rr.buffer[:bufferPtr]
		rr.err = io.EOF
	}

	// we might have over-filled the cache, so evict blocks again
	for int64(len(r.cache)) > r.numBlocks {
		var oldestBlock int64
		oldestTime := r.clock.Load()
		for block, entry := range r.cache {
			lastAccessed := entry.lastAccessed.Load()
			if lastAccessed < oldestTime {
				oldestBlock = block
				oldestTime = lastAccessed
			}
		}
		delete(r.cache, oldestBlock)
	}

	// update stats
	n := len(rr.buffer)
	r.overlayBytes.Add(uint64(n))
	r.overlayReqs.Add(1)
}

func (r *ReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	rr := &readRequest{
		buffer: p,
		offset: off,
	}

	r.cacheMutex.RLock()
	inCache := r.canBeServedFromCache(rr)
	if inCache {
		r.processReadRequest(rr)
		r.cacheMutex.RUnlock()
	} else {
		r.cacheMutex.RUnlock()
		r.cacheMutex.Lock()
		r.processReadRequest(rr)
		if r.prevLine != "" {
			r.drawGraph()
		}
		r.cacheMutex.Unlock()
	}

	return len(rr.buffer), rr.err
}

func (r *ReaderAt) Stats() (
	overlayBytes, underlayBytes, overlayReqs, underlayReqs uint64,
) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	overlayBytes = r.overlayBytes.Load()
	underlayBytes = r.underlayBytes
	overlayReqs = r.overlayReqs.Load()
	underlayReqs = r.underlayReqs
	return
}

// half-open range
func (r *ReaderAt) rangeToBlockRange(start, end int64) (int64, int64) {
	startBlock := start / r.blockSize

	// handle empty range
	if start == end {
		return startBlock, startBlock
	}

	endLocation := end / r.blockSize

	// if end (half-open part) is on a block boundary
	if end%r.blockSize == 0 {
		return startBlock, endLocation
	}

	return startBlock, endLocation + 1
}

func (r *ReaderAt) blockRangeToRange(start, end int64) (int64, int64) {
	return start * r.blockSize, end * r.blockSize
}
