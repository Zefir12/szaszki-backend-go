package internal

import (
	"encoding/binary"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	bh "github.com/zefir/szaszki-go-backend/internal/binaryHelpers"
	"github.com/zefir/szaszki-go-backend/logger"
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
		logger.Log.Info().Uint32("clientId", client.UserID).Uint16("gameMode", gameMode).Msg("Client wants to find game")
		EnqueuePlayerForMode(client, gameMode)
	case ClientCmds.CloseSocket:
		logger.Log.Info().Uint32("clientId", client.UserID).Msg("Client wants to close socket")
	case ClientCmds.MovePiece:
		invalid := func() {
			client.WriteMsg(ServerCmds.InvalidMove, nil)
		}
		logger.Log.Info().Uint32("clientId", client.UserID).Msg("Received move")
		if len(payload) < 3 {
			logger.Log.Warn().Uint32("clientId", client.UserID).Msg("Invalid move payload length")
			invalid()
			return
		}

		ints, err := bh.Unpack(payload, []bh.FieldType{bh.Int8, bh.Int8, bh.Int8, bh.Uint32})
		if err != nil {
			logger.Log.Warn().Uint32("clientId", client.UserID).Err(err).Msg("Can't unpack move")
			invalid()
			return
		}
		from := ints[0].(int8)
		to := ints[1].(int8)
		promoteTo := ints[2].(int8)

		game, ok := keeper.GetGame(ints[3].(uint32))

		if game == nil || !ok {
			logger.Log.Warn().Uint32("clientId", client.UserID).Uint32("gameId", ints[3].(uint32)).Msg("Couldnt find active game with given id")
			invalid()
			return
		}

		move := PlayerMove{
			From:      from,
			To:        to,
			PromoteTo: promoteTo,
			Player:    client,
		}
		logger.Log.Info().Uint32("gameId", game.ID).Int("from", int(from)).Int("to", int(to)).Int("promoteTo", int(promoteTo)).Uint32("playerId", client.UserID).Msg("Sending move to game")
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
			logger.Log.Warn().Err(err).Uint64("connId", id).Msg("WriteMsg error on connection")
		}
	}
	return nil
}
