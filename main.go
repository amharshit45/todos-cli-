package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/amharshit45/todos-cli-/cli"
	"github.com/amharshit45/todos-cli-/storage"
)

func main() {
	_ = godotenv.Load()

	mongoURI := os.Getenv("MONGO_URI")
	mongoDB := os.Getenv("MONGO_DB")
	if mongoURI == "" || mongoDB == "" {
		log.Fatal("MONGO_URI and MONGO_DB must be set in environment")
	}

	store, err := storage.NewMongoStorage(mongoURI, mongoDB)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer store.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		fmt.Println("\nShutting down...")
		os.Stdin.Close()
	}()

	scanner := bufio.NewScanner(os.Stdin)
	app := cli.New(ctx, store, scanner)

	if err := app.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
