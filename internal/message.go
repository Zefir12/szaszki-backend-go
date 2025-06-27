package internal

import (
	"encoding/binary"
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	bh "github.com/zefir/szaszki-go-backend/internal/binaryHelpers"
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
	MoveHappend          MsgType
	InvalidMove          MsgType
	GameState            MsgType
}{
	Ping:                 1,
	OutMsgUpdateVariable: 2,
	ClientAuthenticated:  3,
	GameFound:            4,
	GameStarted:          5,
	GameDeclined:         6,
	GameSearchTimeout:    7,
	MoveHappend:          15,
	InvalidMove:          16,
	GameState:            20,
}

var ClientCmds = struct {
	Pong             MsgType
	Auth             MsgType
	SearchingForGame MsgType
	AcceptedGame     MsgType
	DeclinedGame     MsgType
	CloseSocket      MsgType
	MovePiece        MsgType
}{
	Pong:             1,
	Auth:             2,
	SearchingForGame: 3,
	AcceptedGame:     4,
	DeclinedGame:     5,
	MovePiece:        10,
	CloseSocket:      61500,
}

func handleMessage(msgType MsgType, payload []byte, client *Client) {
	switch msgType {
	case ClientCmds.Pong:
		//connection alive
	case ClientCmds.SearchingForGame:
		gameMode := binary.BigEndian.Uint16(payload)
		log.Println("user with id", client.UserID, "wants to find game with type:", gameMode)
		EnqueuePlayerForMode(client, gameMode)
	case ClientCmds.CloseSocket:
		log.Println("clients wants to close socket")
	case ClientCmds.MovePiece:
		invalid := func() {
			client.WriteMsg(ServerCmds.InvalidMove, nil)
		}
		log.Println("received move")
		if len(payload) < 3 {
			log.Println("invalid move payload length")
			invalid()
			return
		}

		ints, err := bh.Unpack(payload, []bh.FieldType{bh.Int8, bh.Int8, bh.Int8, bh.Uint32})
		if err != nil {
			log.Println("cant unpack move")
			invalid()
			return
		}
		from := ints[0].(int8)
		to := ints[1].(int8)
		promoteTo := ints[2].(int8)

		game, ok := keeper.GetGame(ints[3].(uint32))

		if game == nil || !ok {
			log.Println("client not in any game")
			invalid()
			return
		}

		move := PlayerMove{
			From:      from,
			To:        to,
			PromoteTo: promoteTo,
			Player:    client,
		}
		log.Println(move, "sending move to game")
		game.MoveChannel <- move

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
