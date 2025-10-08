package storage

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/riverqueue/river"
	"github.com/typesense/typesense-go/v2/typesense"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)

type Database struct {
	Postgresql *gorm.DB
	Redis      *redis.Client
	Minio      *minio.Client
	TypeSense  *typesense.Client
	Elastic    *elasticsearch.Client
	Mongo      *mongo.Client
	River      *river.Client[pgx.Tx]
}

var (
	DB     *Database = &Database{}
	Logger *utility.Logger
)

func Connection() *Database {
	return DB
}
