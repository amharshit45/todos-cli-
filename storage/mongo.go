package storage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/amharshit45/todos-cli-/todo"
)

const (
	collectionName    = "todos"
	counterCollection = "counters"
	defaultTimeout    = 5 * time.Second
	listTimeout       = 10 * time.Second
)

var _ todo.Storage = (*MongoStorage)(nil)

type MongoStorage struct {
	client    *mongo.Client
	dbName    string
	closeOnce sync.Once
}

func NewMongoStorage(ctx context.Context, uri, dbName string) (*MongoStorage, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, defaultTimeout)
	defer pingCancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer disconnectCancel()
		client.Disconnect(disconnectCtx)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return &MongoStorage{client: client, dbName: dbName}, nil
}

func (ms *MongoStorage) coll() *mongo.Collection {
	return ms.client.Database(ms.dbName).Collection(collectionName)
}

func (ms *MongoStorage) nextID(ctx context.Context) (int, error) {
	type counter struct {
		Seq int `bson:"seq"`
	}

	var result counter
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	err := ms.client.Database(ms.dbName).Collection(counterCollection).
		FindOneAndUpdate(ctx,
			bson.D{{Key: "_id", Value: collectionName}},
			bson.D{{Key: "$inc", Value: bson.D{{Key: "seq", Value: 1}}}},
			opts,
		).Decode(&result)
	if err != nil {
		return 0, fmt.Errorf("failed to generate next id: %w", err)
	}

	return result.Seq, nil
}

func (ms *MongoStorage) Add(ctx context.Context, description string) error {
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	opCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	id, err := ms.nextID(opCtx)
	if err != nil {
		return err
	}

	newTodo := todo.Todo{ID: id, Description: description}
	if _, err := ms.coll().InsertOne(opCtx, newTodo); err != nil {
		ms.rollbackID(opCtx)
		return fmt.Errorf("failed to insert todo: %w", err)
	}
	return nil
}

func (ms *MongoStorage) rollbackID(ctx context.Context) {
	err := ms.client.Database(ms.dbName).Collection(counterCollection).
		FindOneAndUpdate(ctx,
			bson.D{{Key: "_id", Value: collectionName}},
			bson.D{{Key: "$inc", Value: bson.D{{Key: "seq", Value: -1}}}},
		).Err()
	if err != nil {
		log.Printf("failed to rollback ID counter: %v", err)
	}
}

func (ms *MongoStorage) List(ctx context.Context) ([]todo.Todo, error) {
	opCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	cursor, err := ms.coll().Find(opCtx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find todos: %w", err)
	}

	var todos []todo.Todo
	if err := cursor.All(opCtx, &todos); err != nil {
		return nil, fmt.Errorf("failed to decode todos: %w", err)
	}
	if todos == nil {
		todos = []todo.Todo{}
	}
	return todos, nil
}

func (ms *MongoStorage) Delete(ctx context.Context, id int) error {
	if err := todo.ValidateID(id); err != nil {
		return err
	}
	opCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	result, err := ms.coll().DeleteOne(opCtx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
	}
	return nil
}

func (ms *MongoStorage) SetCompleted(ctx context.Context, id int, completed bool) error {
	if err := todo.ValidateID(id); err != nil {
		return err
	}
	opCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	result, err := ms.coll().UpdateOne(opCtx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "completed", Value: completed}}}},
	)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
	}
	if result.ModifiedCount == 0 {
		if completed {
			return fmt.Errorf("todo %d: %w", id, todo.ErrAlreadyCompleted)
		}
		return fmt.Errorf("todo %d: %w", id, todo.ErrAlreadyIncomplete)
	}
	return nil
}

func (ms *MongoStorage) Edit(ctx context.Context, id int, description string) error {
	if err := todo.ValidateID(id); err != nil {
		return err
	}
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	opCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	result, err := ms.coll().UpdateOne(opCtx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "description", Value: description}}}},
	)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
	}
	return nil
}

func (ms *MongoStorage) Close(ctx context.Context) error {
	var err error
	ms.closeOnce.Do(func() {
		err = ms.client.Disconnect(ctx)
	})
	return err
}
