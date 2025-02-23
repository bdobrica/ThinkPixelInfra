package parquet

import (
	"context"
	"snap_store/logger"
	"snap_store/parser"

	"github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/writer"
)

// WriteParquet writes parsed documents to a Parquet file.
func WriteParquet(ctx context.Context, bucket, key string, docs []parser.ParsedDocument) error {
	fw, err := s3.NewS3FileWriter(ctx, bucket, key, nil)
	if err != nil {
		logger.Errorf("Error creating S3 file writer: %v", err)
		return err
	}
	defer fw.Close()

	pw, err := writer.NewParquetWriter(fw, new(parser.ParsedDocument), 4)
	if err != nil {
		logger.Errorf("Error initializing Parquet writer: %v", err)
		return err
	}
	defer pw.WriteStop()

	for _, doc := range docs {
		if err := pw.Write(doc); err != nil {
			logger.Errorf("Error writing document to Parquet: %v", err)
			return err
		}
	}

	logger.Infof("Successfully wrote %d documents to Parquet file: s3://%s/%s", len(docs), bucket, key)
	return nil
}
