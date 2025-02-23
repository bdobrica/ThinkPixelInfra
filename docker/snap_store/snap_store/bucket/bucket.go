package bucket

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// BucketClient wraps the MinIO client and configuration
type BucketClient struct {
	Client     *minio.Client
	BucketName string
	ClusterID  string
}

// ObjectInfo wraps the MinIO object info
type ObjectInfo minio.ObjectInfo

// NewBucketClient initializes a new BucketClient
func NewBucketClient(endpoint, accessKeyID, secretAccessKey, bucketName, clusterID string, useSSL bool) (*BucketClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &BucketClient{Client: client, BucketName: bucketName, ClusterID: clusterID}, nil
}

// ListObjects lists objects in the specified bucket with the given prefix
func (bc *BucketClient) ListObjects(prefix string) ([]ObjectInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	objectCh := bc.Client.ListObjects(ctx, bc.BucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	var objects []ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		object := ObjectInfo(object)
		objects = append(objects, object)
	}
	return objects, nil
}

func (bc *BucketClient) GetObject(ctx context.Context, objectName string) (*minio.Object, error) {
	object, err := bc.Client.GetObject(ctx, bc.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return object, nil
}

// DownloadObject downloads an object from the bucket
func (bc *BucketClient) DownloadObject(objectName string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	object, err := bc.GetObject(ctx, objectName)
	if err != nil {
		return nil, err
	}
	defer object.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(object); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SaveToFile saves the downloaded object to a local file
func SaveToFile(data []byte, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// ListPodNames lists all pod names dynamically
func (bc *BucketClient) ListPodNames(ctx context.Context, prefix string) ([]string, error) {
	objects, err := bc.ListObjects(prefix)
	if err != nil {
		return nil, err
	}

	var podNames []string
	for _, object := range objects {
		if strings.HasSuffix(object.Key, "/") {
			parts := strings.Split(object.Key, "/")
			if len(parts) > 2 {
				podNames = append(podNames, parts[2])
			}
		}
	}
	return podNames, nil
}
