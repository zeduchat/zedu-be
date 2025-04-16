package storage

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/utility"
	"github.com/minio/minio-go/v7"
	"github.com/typesense/typesense-go/v2/typesense"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type Database struct {
	Postgresql *gorm.DB
	Redis      *redis.Client
	Minio      *minio.Client
	TypeSense  *typesense.Client
	Elastic    *elasticsearch.Client
	Mongo      *mongo.Client
}

var (
	DB     *Database = &Database{}
	Logger *utility.Logger
)

func Connection() *Database {
	return DB
}
