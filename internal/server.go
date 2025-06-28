package internal

import (
	"encoding/binary"
	"io"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	authclient "github.com/zefir/szaszki-go-backend/grpc"
	bh "github.com/zefir/szaszki-go-backend/internal/binaryHelpers"
	"github.com/zefir/szaszki-go-backend/logger"
)

var connCounter uint64 = 0
var connCounterMu sync.Mutex

func generateConnID() uint64 {
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

	logger.Log.Info().Str("addr", addr).Msg("WebSocket server started")

	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Log.Warn().Err(err).Msg("Accept error")
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	connID := generateConnID()

	_, err := ws.Upgrade(conn)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("WebSocket upgrade error")
		conn.Close()
		return
	}

	br := wsutil.NewReader(conn, ws.StateServerSide)

	var userID uint32
	var client *Client

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
		if userID == 0 && msgType == ClientCmds.Auth {
			token := string(payload)
			valid, uid, err := authclient.ValidateToken(token)
			if err != nil || !valid {
				logger.Log.Warn().Err(err).Msg("Invalid token")
				conn.Close()
				closeConn(client, connID)
				break
			}
			userID = uid

			// Add or get shared Client for this user
			client = GetClientOrCreate(userID)
			client.AddConn(connID, conn)

			payload, _ := bh.Pack([]bh.FieldType{bh.Uint32}, []any{client.UserID})

			// Send auth success
			WriteMsgToSingleConn(conn, ServerCmds.ClientAuthenticated, payload)

			PutBuffer(bufPtr)
			continue
		}

		if userID == 0 {
			// Not authenticated and not auth message, close connection
			conn.Close()
			closeConn(client, connID)
			break
		}

		// Now handle other messages with the shared client instance
		handleMessage(msgType, payload, client)

		PutBuffer(bufPtr)
	}
	closeConn(client, connID)
}

func closeConn(client *Client, connID uint64) { // Connection closed, remove this connection from the client's map
	if client != nil {
		remainingConns := client.RemoveConn(connID)
		if remainingConns <= 0 {
			RemoveClient(client.UserID)
		}
		logger.Log.Info().Uint32("clientId", client.UserID).Int("remainingConns", remainingConns).Msg("Connection closed, client has remaining connections")
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

//https://go101.org/article/channel.html
