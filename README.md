This package implements an `io.ReaderAt` which wraps annother `io.ReaderAt` with a LRU cache. It has moderate concurrency support in that it is safe for use from multiple goroutines. If a request can be served entirely from the cache it may take place concurrently with other requests, but requests which require fetching data from the underlying `io.Reader` are serialized.

The tuneables for the cache are `blockSize` and `numBlocks`. The total cache size will be `blockSize * numBlocks` large.
```go
// 16x 4KB blocks (64KB total)
buffered := bufreadat.New(underlying, 4*1024, 16)
```

`bufreadat.ReaderAt` additionally exposes a `Stats()` method you can use to see how many bytes and requests were performed vs how many bytes and requests were performed on the underlying reader
```go
overBytes, underBytes, overReqs, underReqs := buffered.Stats()
fmt.Printf(
	"Reader: %d/%d (%.2fx improvement) | Reqs: %d/%d (%.2fx improvement)\n",
	overBytes, underBytes, float64(overBytes)/float64(underBytes),
	overReqs, underReqs, float64(overReqs)/float64(underReqs),
)
```

Because the buffered reader only performes block-aligned reads on the underlying reader, it is possible to have an improvement less than 1.00 (i.e. the cache is actually making things worse). If this happens, try using a smaller block size.

This library is largely inspired by [buf-readerat](https://github.com/avvmoto/buf-readerat)
