package database

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DBClient struct {
	client   *mongo.Client
	Tarjetas *TarjetasRepository
}

func NewDBClient(ctx context.Context, uri, dbName string) (*DBClient, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(dbName)

	return &DBClient{
		client: client,
		Tarjetas: &TarjetasRepository{
			collection: db.Collection("tarjetas"),
		},
	}, nil
}

func (c *DBClient) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}
