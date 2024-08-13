package storage

import (
	"github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type Database struct {
	Postgresql *gorm.DB
	Redis      *redis.Client
	Minio      *minio.Client
}

var DB *Database = &Database{}

func Connection() *Database {
	return DB
}
