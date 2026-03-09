package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	scanner := bufio.NewScanner(os.Stdin)
	app := cli.New(store, scanner, os.Stdout)

	if err := app.Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
