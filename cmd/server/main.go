package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/zefir/szaszki-go-backend/config"
	authclient "github.com/zefir/szaszki-go-backend/grpc"
	"github.com/zefir/szaszki-go-backend/internal"
)

func logRuntimeStats() {
	var m runtime.MemStats
	for {
		runtime.ReadMemStats(&m)
		var clients = internal.GetAllClients()
		log.Printf("[RUNTIME] Alloc = %.2f MiB | Sys = %.2f MiB | NumGC = %v | Goroutines = %d | ClienctConnected = %d",
			float64(m.Alloc)/1024/1024,
			float64(m.Sys)/1024/1024,
			m.NumGC,
			runtime.NumGoroutine(),
			len(clients),
		)
		time.Sleep(5 * time.Second)
	}
}

func main() {
	config.Load()
	go logRuntimeStats()
	authclient.Init("localhost:" + config.AppConfig.GRPC_PORT)

	internal.InitGameKeeper()
	internal.InitAllMatchmakers(100)
	fmt.Println("Server running on port " + config.AppConfig.WS_PORT)
	err := internal.ListenAndServe("localhost:" + config.AppConfig.WS_PORT)
	if err != nil {
		log.Fatal("WebSocket server error:", err)
	}
	fmt.Println("Server closeing")
}

// protoc --go_out=. --go-grpc_out=. proto/auth.proto
