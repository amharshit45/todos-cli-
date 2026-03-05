package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/amharshit45/todos-cli-/todo"
)

const collectionName = "todos"

type MongoStorage struct {
	client *mongo.Client
	dbName string
}

func NewMongoStorage(uri, dbName string) (*MongoStorage, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return &MongoStorage{client: client, dbName: dbName}, nil
}

func (ms *MongoStorage) coll() *mongo.Collection {
	return ms.client.Database(ms.dbName).Collection(collectionName)
}

func (ms *MongoStorage) nextID(ctx context.Context) (int, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "id", Value: -1}})
	var t todo.Todo
	err := ms.coll().FindOne(ctx, bson.D{}, opts).Decode(&t)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 1, nil
		}
		return 0, fmt.Errorf("failed to determine next id: %w", err)
	}
	return t.ID + 1, nil
}

func (ms *MongoStorage) findByID(ctx context.Context, id int) (todo.Todo, error) {
	var t todo.Todo
	err := ms.coll().FindOne(ctx, bson.D{{Key: "id", Value: id}}).Decode(&t)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return t, fmt.Errorf("todo with id %d not found", id)
		}
		return t, fmt.Errorf("failed to find todo: %w", err)
	}
	return t, nil
}

func (ms *MongoStorage) Add(description string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := ms.nextID(ctx)
	if err != nil {
		return err
	}

	newTodo := todo.Todo{ID: id, Description: description}
	if _, err := ms.coll().InsertOne(ctx, newTodo); err != nil {
		return fmt.Errorf("failed to insert todo: %w", err)
	}
	return nil
}

func (ms *MongoStorage) List() ([]todo.Todo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := ms.coll().Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find todos: %w", err)
	}

	var todos []todo.Todo
	if err := cursor.All(ctx, &todos); err != nil {
		return nil, fmt.Errorf("failed to decode todos: %w", err)
	}
	if todos == nil {
		todos = []todo.Todo{}
	}
	return todos, nil
}

func (ms *MongoStorage) Delete(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := ms.coll().DeleteOne(ctx, bson.D{{Key: "id", Value: id}})
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("todo with id %d not found", id)
	}
	return nil
}

func (ms *MongoStorage) SetCompleted(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t, err := ms.findByID(ctx, id)
	if err != nil {
		return err
	}
	if t.Completed {
		return fmt.Errorf("todo %d is already completed", id)
	}

	_, err = ms.coll().UpdateOne(ctx,
		bson.D{{Key: "id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "completed", Value: true}}}},
	)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	return nil
}

func (ms *MongoStorage) SetIncomplete(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t, err := ms.findByID(ctx, id)
	if err != nil {
		return err
	}
	if !t.Completed {
		return fmt.Errorf("todo %d is already incomplete", id)
	}

	_, err = ms.coll().UpdateOne(ctx,
		bson.D{{Key: "id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "completed", Value: false}}}},
	)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	return nil
}

func (ms *MongoStorage) Edit(id int, description string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := ms.coll().UpdateOne(ctx,
		bson.D{{Key: "id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "description", Value: description}}}},
	)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("todo with id %d not found", id)
	}
	return nil
}

func (ms *MongoStorage) Close() error {
	return ms.client.Disconnect(context.Background())
}
