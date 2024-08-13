package storage

import (
	"github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)

type Database struct {
	Postgresql *gorm.DB
	Redis      *redis.Client
	Minio      *minio.Client
}

var (
	DB     *Database = &Database{}
	Logger *utility.Logger
)

func Connection() *Database {
	return DB
}
