package oss

import (
	"bytes"
	"context"
	"logic-grpc-service/telemetry"
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

func UploadFile(ctx context.Context, cfg *config.Config, objectKey string, data []byte) (retErr error) {
	ctx, end := telemetry.Start(ctx, "oss.uploadfile")
	defer func() { end(telemetry.Outcome(retErr)) }()
	bucket, err := getBucket(cfg)
	if err != nil {
		return err
	}
	return bucket.PutObject(objectKey, bytes.NewReader(data), oss.WithContext(ctx))
}

func GetSignedURL(ctx context.Context, cfg *config.Config, objectKey string) (_ string, retErr error) {
	ctx, end := telemetry.Start(ctx, "oss.getsignedurl")
	defer func() { end(telemetry.Outcome(retErr)) }()
	bucket, err := getBucket(cfg)
	if err != nil {
		return "", err
	}
	return bucket.SignURL(objectKey, oss.HTTPGet, int64(time.Hour.Seconds()*2))
}

func DeleteFile(ctx context.Context, cfg *config.Config, objectKey string) (retErr error) {
	ctx, end := telemetry.Start(ctx, "oss.deletefile")
	defer func() { end(telemetry.Outcome(retErr)) }()
	bucket, err := getBucket(cfg)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(objectKey, oss.WithContext(ctx))
}

func DownloadFile(ctx context.Context, cfg *config.Config, objectKey string) (_ []byte, retErr error) {
	ctx, end := telemetry.Start(ctx, "oss.downloadfile")
	defer func() { end(telemetry.Outcome(retErr)) }()
	bucket, err := getBucket(cfg)
	if err != nil {
		return nil, err
	}
	body, err := bucket.GetObject(objectKey, oss.WithContext(ctx))
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
