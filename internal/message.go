package internal

import (
	"encoding/binary"
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type MsgType uint16

var ServerCmds = struct {
	Ping                 MsgType
	OutMsgUpdateVariable MsgType
	ClientAuthenticated  MsgType
	GameFound            MsgType
	GameStarted          MsgType
	GameDeclined         MsgType
	GameSearchTimeout    MsgType
}{
	Ping:                 1,
	OutMsgUpdateVariable: 2,
	ClientAuthenticated:  3,
	GameFound:            4,
	GameStarted:          5,
	GameDeclined:         6,
	GameSearchTimeout:    7,
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

func handleMessage(msgType MsgType, payload []byte, client *Client) {
	switch msgType {
	case ClientCmds.Pong:
		//connection alive
	case ClientCmds.SearchingForGame:
		gameMode := binary.BigEndian.Uint16(payload)
		log.Println("user with id", client.UserID, "wants to find game with type:", gameMode)
		matchmaker, ok := matchmakers[gameMode]
		if !ok {
			log.Printf("No matchmaker for mode %d", gameMode)
			return
		}
		matchmaker.Enqueue(client)
	default:
	}
}

func WriteMsgToSingleConn(conn net.Conn, msgType MsgType, payload []byte) error {
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

func (c *Client) WriteMsg(msgType MsgType, payload []byte) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	for id, conn := range c.Conns {
		err := WriteMsgToSingleConn(conn, msgType, payload)
		if err != nil {
			log.Printf("WriteMsg error on connection %d: %v", id, err)
		}
	}
	return nil
}
