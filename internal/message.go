package internal

import (
	"encoding/binary"
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	authclient "github.com/zefir/szaszki-go-backend/grpc"
)

type MsgType uint16

var ServerCmds = struct {
	Ping                 MsgType
	OutMsgUpdateVariable MsgType
	ClientAuthenticated  MsgType
	GameFound            MsgType
}{
	Ping:                 1,
	OutMsgUpdateVariable: 2,
	ClientAuthenticated:  3,
	GameFound:            4,
}

var ClientCmds = struct {
	Pong             MsgType
	RcvMsgAuth       MsgType
	SearchingForGame MsgType
}{
	Pong:             1,
	RcvMsgAuth:       2,
	SearchingForGame: 3,
}

func handleMessage(conn net.Conn, msgType MsgType, payload []byte, client *ClientConn) {
	if client.UserID == 0 {
		if msgType == ClientCmds.RcvMsgAuth {
			token := string(payload)
			valid, userId, err := authclient.ValidateToken(token)
			if err != nil || !valid {
				log.Println("Invalid token:", err)
				conn.Close()
				return
			}
			client.UserID = userId

			AddClient(client.UserID, conn)

			WriteMsg(conn, ServerCmds.ClientAuthenticated, nil)
		} else {
			conn.Close()
		}

	} else {
		switch msgType {
		case ClientCmds.Pong:
			//connection alive
		case ClientCmds.SearchingForGame:
			log.Println("user with id", client.UserID, "wants to find game with type:", payload[0])
			GetMatchmaker().Enqueue(client)
		default:

		}
	}
}

func WriteMsg(conn net.Conn, msgType MsgType, payload []byte) error {
	full := make([]byte, 2+len(payload))

	binary.BigEndian.PutUint16(full[0:], uint16(msgType))

	// Copy payload
	copy(full[2:], payload)

	// Write the entire message as one WebSocket binary frame
	writer := wsutil.NewWriter(conn, ws.StateServerSide, ws.OpBinary)
	if _, err := writer.Write(full); err != nil {
		return err
	}

	return writer.Flush()
}
