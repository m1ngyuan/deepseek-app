package main

import (
	"runtime"
	"sync"
	"time"

	"github.com/manucorporat/stats"
)

var (
	ips           = stats.New()
	messages      = stats.New()
	users         = stats.New()
	mutexIps      sync.RWMutex
	mutexUsers    sync.RWMutex
	mutexMessages sync.RWMutex
	mutexStats    sync.RWMutex
	savedStats    = make(map[string]uint64)
)

func statsWorker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	var lastMallocs uint64
	var lastFrees uint64
	for range ticker.C {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		mutexStats.Lock()
		// Lock messages counter before accessing it
		// Consider creating a dedicated mutex for messages
		savedStats = map[string]uint64{
			"timestamp":  uint64(time.Now().Unix()),
			"HeapInuse":  stats.HeapInuse,
			"StackInuse": stats.StackInuse,
			"Mallocs":    stats.Mallocs - lastMallocs,
			"Frees":      stats.Frees - lastFrees,
			"Inbound":    uint64(messages.Get("inbound")),
			"Outbound":   uint64(messages.Get("outbound")),
			"Connected":  connectedUsers(),
		}
		lastMallocs = stats.Mallocs
		lastFrees = stats.Frees
		messages.Reset()
		mutexStats.Unlock()
	}
}

func connectedUsers() uint64 {
	connected := users.Get("connected") - users.Get("disconnected")
	if connected < 0 {
		return 0
	}
	return uint64(connected)
}

// Stats returns savedStats data.
func Stats() map[string]uint64 {
	mutexStats.RLock()
	defer mutexStats.RUnlock()

	if savedStats == nil {
		return make(map[string]uint64)
	}
	return savedStats
}
