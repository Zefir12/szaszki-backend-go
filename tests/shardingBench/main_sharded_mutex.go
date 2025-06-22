//go:build sharded

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/zefir/szaszki-go-backend/tests/shardingBench/helpers"
)

type shard struct {
	mu     sync.RWMutex
	values map[uint32]*helpers.Client
}

const shards_amount = 256

type ShardedStore struct {
	shards [shards_amount]shard
}

func NewShardedStore() *ShardedStore {
	s := &ShardedStore{}
	for i := range s.shards {
		s.shards[i].values = make(map[uint32]*helpers.Client)
	}
	return s
}

func (s *ShardedStore) getShard(id uint32) *shard {
	return &s.shards[id%shards_amount]
}

func (s *ShardedStore) Set(id uint32, c *helpers.Client) {
	sh := s.getShard(id)
	sh.mu.Lock()
	sh.values[id] = c
	sh.mu.Unlock()
}

func (s *ShardedStore) Get(id uint32) (*helpers.Client, bool) {
	sh := s.getShard(id)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	val, ok := sh.values[id]
	return val, ok
}

func main() {
	store := NewShardedStore()

	cpuFile, _ := os.Create("tests/shardingBench/results/cpu_sharded_mutex.prof")
	defer cpuFile.Close()
	pprof.StartCPUProfile(cpuFile)

	var memStart, memEnd runtime.MemStats
	runtime.ReadMemStats(&memStart)

	helpers.GenerateConcurrentWorkload(store.Set, store.Get)

	pprof.StopCPUProfile()
	runtime.GC()
	runtime.ReadMemStats(&memEnd)

	memFile, _ := os.Create("tests/shardingBench/results/mem_sharded_mutex.prof")
	defer memFile.Close()
	pprof.WriteHeapProfile(memFile)

	fmt.Printf("ShardedMutex: %.2f MiB -> %.2f MiB\n",
		helpers.BToMb(memStart.Alloc), helpers.BToMb(memEnd.Alloc))
}
