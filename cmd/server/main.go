package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	"github.com/amharshit45/todos-cli-/gen/todopb"
	"github.com/amharshit45/todos-cli-/server"
	"github.com/amharshit45/todos-cli-/storage"
)

func main() {
	_ = godotenv.Load()

	mongoURI := os.Getenv("MONGO_URI")
	mongoDB := os.Getenv("MONGO_DB")
	if mongoURI == "" || mongoDB == "" {
		log.Fatal("MONGO_URI and MONGO_DB must be set in environment")
	}

	listenAddr := os.Getenv("GRPC_ADDR")
	if listenAddr == "" {
		listenAddr = ":50051"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := storage.NewMongoStorage(ctx, mongoURI, mongoDB)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := store.Close(closeCtx); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
	}

	grpcServer := grpc.NewServer()
	todopb.RegisterTodoServiceServer(grpcServer, server.New(store))

	go func() {
		<-ctx.Done()
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("gRPC server listening on %s", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
