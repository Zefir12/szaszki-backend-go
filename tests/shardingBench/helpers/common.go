package helpers

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Client struct {
	ID uint32
}

type Store interface {
	Set(uint32, *Client)
	Get(uint32) (*Client, bool)
}

func GenerateConcurrentWorkload(
	setFunc func(uint32, *Client),
	getFunc func(uint32) (*Client, bool),
) {
	const (
		ops       = 25_000_000
		workers   = 1
		opsPerWkr = ops / workers
	)
	var wg sync.WaitGroup
	wg.Add(workers)

	start := time.Now() // Start timer

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWkr; i++ {
				id := uint32(rand.Intn(1_000_000))
				if i%2 == 0 {
					setFunc(id, &Client{ID: id})
				} else {
					getFunc(id)
				}
			}
		}()
	}

	wg.Wait()

	elapsed := time.Since(start) // Calculate elapsed time
	fmt.Printf("Total runtime: %s\n", elapsed)
}

func BToMb(b uint64) float64 {
	return float64(b) / 1024.0 / 1024.0
}

//go tool pprof -http=:8080 cpu_sharded_mutex.prof
//go tool pprof -http=localhost:8080 mem.prof

// go run -tags=single .
// go run -tags=sharded .
// go run -tags=channel .

//go tool pprof -http=:8080 -diff_base="cpu_single_mutex.prof" cpu_sharded_mutex.prof
//go tool pprof -http=:8080 -diff_base="tests/shardingBench/results/cpu_single_mutex.prof" tests/shardingBench/results/cpu_sharded_mutex.prof
