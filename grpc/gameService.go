package grpc

import (
	"context"
	"log"

	"google.golang.org/grpc"
	pb "github.com/zefir/szaszki-go-backend/grpc/stuff"
)

func SaveGame(gameID uint32, userIDWhite uint32, userIDBlack uint32, gameState *pb.GameState, pgn string) (*pb.SaveGameResponse, error) {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewGameServiceClient(conn)

	req := &pb.SaveGameRequest{
		GameId:      gameID,
		UserIdWhite: userIDWhite,
		UserIdBlack: userIDBlack,
		GameState:   gameState,
		Pgn:         pgn,
	}

	res, err := client.SaveGame(context.Background(), req)
	if err != nil {
		return nil, err
	}

	log.Printf("SaveGame response: %v", res)
	return res, nil
}