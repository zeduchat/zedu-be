package config

type Minio struct {
	MinioEndpoint string
	BucketName    string
	AccessKey     string
	Secret        string
	UseSSL        bool
}
