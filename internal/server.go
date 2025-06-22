package internal

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	authclient "github.com/zefir/szaszki-go-backend/grpc"
)

var connCounter int64 = 0
var connCounterMu sync.Mutex

func generateConnID() int64 {
	connCounterMu.Lock()
	defer connCounterMu.Unlock()
	connCounter++
	return connCounter
}

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
	connID := generateConnID() // e.g., UUID

	_, err := ws.Upgrade(conn)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		conn.Close()
		return
	}

	br := wsutil.NewReader(conn, ws.StateServerSide)

	var userID int32
	var client *Client

	done := make(chan struct{})

	// Ping goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := WriteMsgToSingleConn(conn, ServerCmds.Ping, nil)
				if err != nil {
					log.Println("Ping error:", err)
					close(done)
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		hdr, err := br.NextFrame()
		if err != nil {
			// handle errors and cleanup
			break
		}

		if hdr.OpCode != ws.OpBinary {
			// discard non-binary frames
			if _, err := io.CopyN(io.Discard, br, int64(hdr.Length)); err != nil {
				break
			}
			continue
		}

		size := int(hdr.Length)
		bufPtr := GetBufferForSize(size)
		buf := *bufPtr
		buf = buf[:size]
		_, err = io.ReadFull(br, buf)
		if err != nil {
			PutBuffer(bufPtr)
			break
		}

		if len(buf) < 2 {
			PutBuffer(bufPtr)
			continue
		}

		msgType := MsgType(binary.BigEndian.Uint16(buf[0:2]))
		payload := buf[2:]

		// Handle auth message specially
		if userID == 0 && msgType == ClientCmds.RcvMsgAuth {
			token := string(payload)
			valid, uid, err := authclient.ValidateToken(token)
			if err != nil || !valid {
				log.Println("Invalid token:", err)
				conn.Close()
				break
			}
			userID = uid

			// Add or get shared Client for this user
			client = GetClientOrCreate(userID)
			client.AddConn(connID, conn)

			// Send auth success
			WriteMsgToSingleConn(conn, ServerCmds.ClientAuthenticated, nil)

			PutBuffer(bufPtr)
			continue
		}

		if userID == 0 {
			// Not authenticated and not auth message, close connection
			conn.Close()
			break
		}

		// Now handle other messages with the shared client instance
		handleMessage(msgType, payload, client)

		PutBuffer(bufPtr)
	}

	// Connection closed, remove this connection from the client's map
	if client != nil {
		client.RemoveConn(connID)
		if client.ConnCount() == 0 {
			RemoveClient(userID)
		}
	}
}

// func handleConn(conn net.Conn) {
// 	_, err := ws.Upgrade(conn)
// 	if err != nil {
// 		log.Println("WebSocket upgrade error:", err)
// 		conn.Close()
// 		return
// 	}
// 	defer conn.Close()
// 	buf := make([]byte, 6000) // max buffer size
// 	for {

// 		n, err := conn.Read(buf)
// 		if err != nil {
// 			if err == io.EOF {
// 				return
// 			}
// 			if strings.Contains(err.Error(), "wsarecv") {
// 				return
// 			}
// 			log.Println("Frame read error:", err)
// 		}

// 		if n < 2 {
// 			log.Println("Received too few bytes to parse MsgType")
// 			continue
// 		}

// 		msgType := MsgType(binary.BigEndian.Uint16(buf[0:2]))
// 		payload := buf[2:n]

// 		handleMessage(conn, msgType, payload)
// 	}
// }
