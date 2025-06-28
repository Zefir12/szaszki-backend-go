package logger

import (
	"context"
	"time"

	pb "github.com/zefir/szaszki-go-backend/grpc/stuff"

	"github.com/rs/zerolog"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCLogger struct {
	client  pb.LogServiceClient
	service string
}

func NewGRPCLogger(conn *grpc.ClientConn, service string) *GRPCLogger {
	client := pb.NewLogServiceClient(conn)
	return &GRPCLogger{client: client, service: service}
}

func (g *GRPCLogger) Send(level, msg string, metadata map[string]string) {
	req := &pb.LogRequest{
		Service:   g.service,
		Level:     level,
		Message:   msg,
		Timestamp: timestamppb.Now(),
		Metadata:  metadata,
	}

	// Non-blocking fire-and-forget
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = g.client.SendLog(ctx, req) // ignore error or log locally
	}()
}

type GRPCWriter struct {
	sender *GRPCLogger
}

func NewGRPCWriter(sender *GRPCLogger) GRPCWriter {
	return GRPCWriter{sender: sender}
}

func (w GRPCWriter) Write(p []byte) (n int, err error) {
	// Zerolog sends JSON log as []byte, we extract message & metadata if needed
	msg := string(p)
	w.sender.Send("info", msg, nil)
	return len(p), nil
}

var Log zerolog.Logger
