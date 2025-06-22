//go:build channel

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/zefir/szaszki-go-backend/tests/shardingBench/helpers"
)

type ChannelManager struct {
	mu     sync.RWMutex
	values map[uint32]*helpers.Client
}

func NewChannelManager() *ChannelManager {
	return &ChannelManager{values: make(map[uint32]*helpers.Client)}
}

func (c *ChannelManager) Set(id uint32, cl *helpers.Client) {
	c.mu.Lock()
	c.values[id] = cl
	c.mu.Unlock()
}

func (c *ChannelManager) Get(id uint32) (*helpers.Client, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.values[id]
	return val, ok
}

func main() {
	store := NewChannelManager()

	cpuFile, _ := os.Create("tests/shardingBench/results/cpu_channel_manager.prof")
	defer cpuFile.Close()
	pprof.StartCPUProfile(cpuFile)

	var memStart, memEnd runtime.MemStats
	runtime.ReadMemStats(&memStart)

	helpers.GenerateConcurrentWorkload(store.Set, store.Get)

	pprof.StopCPUProfile()
	runtime.GC()
	runtime.ReadMemStats(&memEnd)

	memFile, _ := os.Create("tests/shardingBench/results/mem_channel_manager.prof")
	defer memFile.Close()
	pprof.WriteHeapProfile(memFile)

	fmt.Printf("ChannelManager: %.2f MiB -> %.2f MiB\n",
		helpers.BToMb(memStart.Alloc), helpers.BToMb(memEnd.Alloc))
}
