package internal

import (
	"encoding/binary"
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	bh "github.com/zefir/szaszki-go-backend/internal/binaryHelpers"
	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
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
	case ClientCmds.AcceptedGame:
		matchID := binary.BigEndian.Uint32(payload[0:4])
		mode := binary.BigEndian.Uint16(payload[4:6])
		AcceptMatch(client, matchID, true, mode)
		log.Println("match accepted", matchID, mode, client.UserID)
	case ClientCmds.DeclinedGame:
		matchID := binary.BigEndian.Uint32(payload[0:4])
		mode := binary.BigEndian.Uint16(payload[4:6])
		AcceptMatch(client, matchID, false, mode)
		log.Println("match declined", matchID, mode, client.UserID)
	case ClientCmds.CloseSocket:

	case ClientCmds.MovePiece:
		invalid := func() {
			client.WriteMsg(ServerCmds.InvalidMove, nil)

		}
		if len(payload) < 2 {
			log.Println("invalid move payload length")
			invalid()
			return
		}
		from := int8(payload[0])
		to := int8(payload[1])

		gameId, err := bh.Unpack(payload, []bh.FieldType{bh.Uint32})
		if err != nil {
			log.Println("client not in any game")
			invalid()
			return
		}
		game, ok := keeper.GetGame(gameId[0].(uint32))

		if game == nil || !ok {
			log.Println("client not in any game")
			invalid()
			return
		}

		// Make sure it's this player's turn:
		if game.Players[game.SideToMove] != client {
			log.Println("not player's turn", client.UserID)
			invalid()
			return
		}

		// Check legality and apply move
		if !chess.IsMoveLegal(&game.Board, from, to) {
			log.Println("illegal move by", client.UserID)
			invalid()
			return
		}

		chess.MakeMove(&game.Board, from, to)
		game.SideToMove = 1 - game.SideToMove

		// Broadcast updated game state or move to all players
		game.BroadcastMove(from, to)
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
