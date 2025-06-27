package grpc

import (
	"context"
	"log"
	"time"

	pb "github.com/zefir/szaszki-go-backend/grpc/stuff"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var client pb.AuthServiceClient

// Init connects to gRPC auth server and sets the client
func Init(addr string) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[authclient] Failed to connect: %v", err)
	}
	client = pb.NewAuthServiceClient(conn)
}

// ValidateToken calls gRPC ValidateToken method
func ValidateToken(token string) (bool, uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.ValidateToken(ctx, &pb.TokenRequest{Token: token})
	if err != nil {
		return false, 0, err
	}
	return resp.Valid, resp.UserId, nil
}

func SendGoServerStats(clientsConnected int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.SendGoServerStats(ctx, &pb.GoServerStats{WsClientsConnected: clientsConnected})
	if err != nil {
		return err
	}
	return nil
}
