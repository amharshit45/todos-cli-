package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/amharshit45/todos-cli-/cli"
	"github.com/amharshit45/todos-cli-/grpcclient"
)

func main() {
	_ = godotenv.Load()

	serverAddr := os.Getenv("GRPC_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:50051"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server at %s: %v", serverAddr, err)
	}

	store := grpcclient.NewStorage(conn)
	defer func() {
		if err := store.Close(ctx); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	app := cli.New(store, scanner, os.Stdout)

	if err := app.Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
