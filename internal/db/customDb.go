package db

/*
TODO: Implement the customDb in this file.

You can change the file name if you want.

Make sure your db implement the DbIf interface in db.go.

Since the customDb will not test in TA's environment, you can just harcode the db's configuration at this file, e.g. db address/keys.
*/

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	mongoURI      = "mongodb://127.0.0.1:27017"
	mongoDBName   = "nsf"
	mongoCollName = "services"
)

type customDb struct {
	client *mongo.Client
	coll   *mongo.Collection
}

// Compile-time check: customDb must implement DbIf.
var _ DbIf = (*customDb)(nil)

func newCustomDb() (*customDb, error) {
	// In driver v2, Connect does not take a context.
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	// Ping to make sure the server is actually reachable.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return &customDb{
		client: client,
		coll:   client.Database(mongoDBName).Collection(mongoCollName),
	}, nil
}

func (c *customDb) Release() error {
	if c.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Disconnect(ctx)
}

func (c *customDb) Save(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Upsert: update the document if _id exists, otherwise insert it.
	filter := bson.M{"_id": key}
	update := bson.M{"$set": bson.M{"value": value}}

	_, err := c.coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("save %s: %w", key, err)
	}
	return nil
}

func (c *customDb) SaveAll(data map[string]string) error {
	// MongoDB has no multi-document atomicity by default, so write one by one.
	for key, value := range data {
		if err := c.Save(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (c *customDb) Exist(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := c.coll.CountDocuments(ctx, bson.M{"_id": key})
	if err != nil {
		return false, fmt.Errorf("exist %s: %w", key, err)
	}
	return count > 0, nil
}

func (c *customDb) Load(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var doc struct {
		Value string `bson:"value"`
	}

	err := c.coll.FindOne(ctx, bson.M{"_id": key}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Match bbolt behavior: a missing key returns an empty string and no error.
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load %s: %w", key, err)
	}
	return doc.Value, nil
}

func (c *customDb) LoadAll() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := c.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("load all: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var docs []struct {
		ID    string `bson:"_id"`
		Value string `bson:"value"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode all: %w", err)
	}

	result := make(map[string]string, len(docs))
	for _, doc := range docs {
		result[doc.ID] = doc.Value
	}
	return result, nil
}

func (c *customDb) Remove(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// DeleteOne does not return an error when the key does not exist.
	_, err := c.coll.DeleteOne(ctx, bson.M{"_id": key})
	if err != nil {
		return fmt.Errorf("remove %s: %w", key, err)
	}
	return nil
}

func (c *customDb) RemoveAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// An empty filter matches every document in the collection.
	_, err := c.coll.DeleteMany(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("remove all: %w", err)
	}
	return nil
}
