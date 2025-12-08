package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	adminServer "github.com/zefir/szaszki-go-backend/admin"
	"github.com/zefir/szaszki-go-backend/config"
	authclient "github.com/zefir/szaszki-go-backend/grpc"
	"github.com/zefir/szaszki-go-backend/internal"
	"github.com/zefir/szaszki-go-backend/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func startHealthServer() {
	http.HandleFunc("/health", healthHandler)
	healthPort := "3131"

	log.Println("Health server listening on", healthPort)
	if err := http.ListenAndServe("0.0.0.0:"+healthPort, nil); err != nil {
		log.Fatalf("Health server failed: %v", err)
	}
}

func main() {
	cfg, err := config.Instance.Get()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect to Node gRPC log service
	conn, err := grpc.NewClient(cfg.NODE_GRPC_ADDR+":"+strconv.Itoa(int(cfg.GRPC_PORT)), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic("Failed to connect to gRPC logger: " + err.Error())
	}
	defer conn.Close()

	// Setup global logger
	logSender := logger.NewGRPCLogger(conn, "go-backend")
	writer := logger.NewGRPCWriter(logSender)
	logger.Log = zerolog.New(writer).With().Timestamp().Logger()

	logger.Log.Info().Str("status", "booted").Msg("Szaszki server starting up")

	// Start healthcheck server in the background
	go startHealthServer()
	go adminServer.StartAdminServer()

	//go logRuntimeStats()
	authclient.Init(conn)

	internal.InitGameKeeper()
	internal.InitAllMatchmakers(100)
	fmt.Println("Server running on port " + strconv.Itoa(int(cfg.WS_PORT)))
	logger.Log.Info().Str("status", "running").Msg("Server started")
	lerr := internal.ListenAndServe("0.0.0.0:" + strconv.Itoa(int(cfg.WS_PORT)))
	if lerr != nil {
		log.Fatal("WebSocket server error:", err)
		logger.Log.Panic().Str("status", "error").Msg("Somthing went wrong with server startup")
	}
	fmt.Println("Server closeing")
}

// protoc --go_out=. --go-grpc_out=. proto/auth.proto
