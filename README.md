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

For debugging purposes, `bufreadat.ReaderAt` exposes `EnableGraph(fileLen)`, which will print a graph to stdout representing the cache every time a call is made to the underlying `io.ReaderAt`. It it probably not suitable for programmatic use, since it assumes it is the only thing drawing to the terminal, but it is a quick and dirty way to see what your reads look like. In order to scale the graph properly, it needs to know the length of the underlying reader. Here is an example of several sequential reads followed by several random reads
```txt
=====================================================================
⣿⣿⣶⣶⣤⣤⣀⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣶⣦⣤⣤⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠛⠛⠿⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣶⣦⣤⣄⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠉⠉⠙⠛⠻⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣶⣤⣤⣀⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠉⠛⠛⠿⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣶⣦⣤⣄⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠙⠛⠛⠿⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣶⣤⣤⣄⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠉⠙⠛⠻⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣶⣶⣤⣤⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠛⠛⠿⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣶⣦⣤⣄⣀⣀
⣀⣀⣿⣿⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣶⣶⣶⣶⣶⣶⡆⠀⢠⣤⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠉⠉⠉⣿⣿⣿⣿⣿⣿⡟⠛⠻⠿⠿
⠛⠛⠿⠿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⠉⠉⠁⠀⠘⠛⢣⣤⣤⣤⣤⣤⣤⣶⣶⣶⣶⣆⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠉⠙⠛⠃⠀⠀⠀⠀
⣶⣶⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣀⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠉⠛⠛⠀⠀⢠⣤⣤⣤⡜⠛⢻⣿⣿⣿⣿⣿⣿⠿⠿⠿⠿⠇⠀⠀⠀⠀⣿⣿⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀
⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣸⣿⣿⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠿⠿⣿⣿⣿⣿⡟⠛⠛⠛⠃⠀⢨⣭⣭⣭⣭⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣶⣶⣶⣶⡆⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⣤⣤⣤⣤⣼⣿⡿⠿⠿⠿⠿⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⣿⣿⣿⣛⣛⣛⣛⡃⠀⠀⠀⠀⠀⠈⠉⠉⠉⠉⣶⣶⣶⣶⣶⣶⡆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠁⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠿⠿⢿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣀⣿⣿⣿⣿⣿⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣭⣭⣿⣿⡟⠛⠃⠀⠀⠀⠀⠀⠀⠀⠀⣶⣶⡆⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⠛⠻⠿⠿⠀⠀⣤⣤⣤⣤⡄⠀⠀⠀⢸⣿⣿⣿⣿⠉⠉⠛⠛⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⣿⣇⣀⡀⠀⢰⣶⣶⠀⠀⣿⣿⣿⣿⡇⠀⠀⠀⠀⠀⠀
⣿⣿⣿⣿⣤⣤⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⣿⣇⣀⣰⣶⡎⠉⠉⠛⠛⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⡇⠀⢸⣿⣿⠀⠀⠿⠿⣿⣿⡇⠀⠀⠀⠀⠀⠀
⠉⠉⠿⠿⠿⠿⠀⠀⠀⠀⢠⣤⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⠛⢳⣶⣶⣶⣶⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⣀⣀⣀⣀⡀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⠿⠿⠿⠿⠉⠉⠀⠀⠉⠉⠉⠉⠁⠀⠘⠛⠛⠿⠿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠶⠶⠶⠶⠶⠶⠆⠀⠀⠀⠀⠿⠿⠿⠿⠿⠿⠧⠤⠄⠀⠀
```

Because the buffered reader only performes block-aligned reads on the underlying reader, it is possible to have an improvement less than 1.00 (i.e. the cache is actually making things worse). If this happens, try using a smaller block size.

This library is largely inspired by [buf-readerat](https://github.com/avvmoto/buf-readerat)
