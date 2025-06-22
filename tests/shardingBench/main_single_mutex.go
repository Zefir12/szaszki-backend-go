//go:build single

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/zefir/szaszki-go-backend/tests/shardingBench/helpers"
)

type SingleMutexStore struct {
	mu     sync.RWMutex
	values map[uint32]*helpers.Client
}

func NewSingleMutexStore() *SingleMutexStore {
	return &SingleMutexStore{values: make(map[uint32]*helpers.Client)}
}

func (s *SingleMutexStore) Set(id uint32, c *helpers.Client) {
	s.mu.Lock()
	s.values[id] = c
	s.mu.Unlock()
}

func (s *SingleMutexStore) Get(id uint32) (*helpers.Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.values[id]
	return val, ok
}

func main() {
	store := NewSingleMutexStore()

	cpuFile, _ := os.Create("tests/shardingBench/results/cpu_single_mutex.prof")
	defer cpuFile.Close()
	pprof.StartCPUProfile(cpuFile)

	var memStart, memEnd runtime.MemStats
	runtime.ReadMemStats(&memStart)

	helpers.GenerateConcurrentWorkload(store.Set, store.Get)

	pprof.StopCPUProfile()
	runtime.GC()
	runtime.ReadMemStats(&memEnd)

	memFile, _ := os.Create("tests/shardingBench/results/mem_single_mutex.prof")
	defer memFile.Close()
	pprof.WriteHeapProfile(memFile)

	fmt.Printf("SingleMutex: %.2f MiB -> %.2f MiB\n",
		helpers.BToMb(memStart.Alloc), helpers.BToMb(memEnd.Alloc))
}
