package main

import (
	"fmt"
	"log"

	"github.com/zefir/szaszki-go-backend/config"
	"github.com/zefir/szaszki-go-backend/internal/db"
	"github.com/zefir/szaszki-go-backend/internal/ws"
)

func main() {
    config.Load()

    if err := db.Init(); err != nil {
        log.Fatal("Database initialization failed:", err)
    }
    defer db.Close()

    fmt.Println("Server running on port " + config.AppConfig.WS_PORT)
    err := ws.ListenAndServe(":" + config.AppConfig.WS_PORT)
    if err != nil {
        log.Fatal("WebSocket server error:", err)
    }
}
