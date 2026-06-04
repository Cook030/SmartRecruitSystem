package oss

import (
	"bytes"
	"time"

	"logic-grpc-service/config"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func getBucket(cfg *config.Config) (*oss.Bucket, error) {
	client, err := oss.New(cfg.OSS.Endpoint, cfg.OSS.AccessKeyID, cfg.OSS.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	return client.Bucket(cfg.OSS.BucketName)
}

func UploadFile(cfg *config.Config, objectKey string, data []byte) error {
	bucket, err := getBucket(cfg)
	if err != nil {
		return err
	}
	return bucket.PutObject(objectKey, bytes.NewReader(data))
}

func GetSignedURL(cfg *config.Config, objectKey string) (string, error) {
	bucket, err := getBucket(cfg)
	if err != nil {
		return "", err
	}
	return bucket.SignURL(objectKey, oss.HTTPGet, int64(time.Hour.Seconds()*2))
}

func DeleteFile(cfg *config.Config, objectKey string) error {
	bucket, err := getBucket(cfg)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(objectKey)
}

func DownloadFile(cfg *config.Config, objectKey string) ([]byte, error) {
	bucket, err := getBucket(cfg)
	if err != nil {
		return nil, err
	}
	body, err := bucket.GetObject(objectKey)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
