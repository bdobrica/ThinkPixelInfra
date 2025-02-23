package parquet

import (
	"context"
	"snap_store/logger"
	"snap_store/parser"

	"github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/reader"
)

// ReadParquet reads parsed documents from a Parquet file.
func ReadParquet(ctx context.Context, bucket, key string) ([]parser.ParsedDocument, error) {
	pf, err := s3.NewS3FileReader(ctx, bucket, key)
	if err != nil {
		logger.Errorf("Error creating S3 file reader: %v", err)
		return nil, err
	}
	defer pf.Close()

	pr, err := reader.NewParquetReader(pf, new(parser.ParsedDocument), 4)
	if err != nil {
		logger.Errorf("Error initializing Parquet reader: %v", err)
		return nil, err
	}
	defer pr.ReadStop()

	var docs []parser.ParsedDocument
	if err := pr.Read(&docs); err != nil {
		logger.Errorf("Error reading documents from Parquet: %v", err)
		return nil, err
	}

	logger.Infof("Successfully read %d documents from Parquet file: s3://%s/%s", len(docs), bucket, key)
	return docs, nil
}
