package ws

import (
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func ListenAndServe(addr string) error {
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }

    log.Println("WebSocket server started on", addr)

    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Println("Accept error:", err)
            continue
        }

        go handleConn(conn)
    }
}

func handleConn(conn net.Conn) {
    _, err := ws.Upgrade(conn)
    if err != nil {
        log.Println("WebSocket upgrade error:", err)
        conn.Close()
        return
    }
    defer conn.Close()

    for {
        msg, op, err := wsutil.ReadClientData(conn)
        if err != nil {
            log.Println("Read error:", err)
            break
        }

        log.Printf("Received: %s", string(msg))

        err = wsutil.WriteServerMessage(conn, op, msg)
        if err != nil {
            log.Println("Write error:", err)
            break
        }
    }
}