package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"snap_store/bucket"
	"snap_store/config"
	"snap_store/logger"
	"snap_store/parser"
)

func main() {
	ctx := context.Background()

	// Define a flag for the date and time
	dateTimeStr := flag.String("datetime", "", "The date and time in RFC3339 format (e.g., 2023-10-01T15:04:05Z)")
	flag.Parse()

	// Parse the date and time if provided
	var now time.Time
	var err error
	if *dateTimeStr != "" {
		now, err = time.Parse(time.RFC3339, *dateTimeStr)
		if err != nil {
			logger.Fatalf("Error parsing date and time: %v", err)
		}
	} else {
		now = time.Now().UTC()
	}

	// Initialize BucketClient
	endpoint := config.GetEnv("SNAP_STORE_S3_ENDPOINT_URL", "fsn1.your-objectstorage.com")
	accessKeyID := config.GetEnv("SNAP_STORE_S3_ACCESS_KEY_ID", "")
	secretAccessKey := config.GetEnv("SNAP_STORE_S3_SECRET_ACCESS_KEY", "")
	bucketName := config.GetEnv("SNAP_STORE_S3_BUCKET_NAME", "dev-thinkpixel-request-logs")
	clusterID := config.GetEnv("SNAP_STORE_CLUSTER_ID", "dev.thinkpixel.io")
	useSSLStr := strings.ToLower(config.GetEnv("SNAP_STORE_S3_USE_SSL", "true"))
	useSSL := useSSLStr == "true" || useSSLStr == "1" || useSSLStr == "yes" || useSSLStr == "on"

	bc, err := bucket.NewBucketClient(endpoint, accessKeyID, secretAccessKey, bucketName, clusterID, useSSL)
	if err != nil {
		logger.Fatalf("Error initializing BucketClient: %v", err)
	}

	// Determine the previous hour (UTC)
	prevHourTime := now.Add(-1 * time.Hour)
	dateStr := prevHourTime.Format("2006-01-02")
	hourStr := prevHourTime.Format("15")

	// Detect pod names using the parser logic.
	podNames, err := parser.DetectPodNames(ctx, bc)
	if err != nil {
		logger.Fatalf("Error detecting pod names: %v", err)
	}

	// For each pod, process logs.
	for _, podName := range podNames {
		// Construct the prefix for this pod.
		prefix := fmt.Sprintf("logs/%s/%s/%s/%s/", bc.ClusterID, podName, dateStr, hourStr)
		logger.Infof("Processing logs with prefix: %s", prefix)

		// Parse logs for this pod.
		parsedDocs, err := parser.ParseLogs(ctx, bc, prefix)
		if err != nil {
			logger.Errorf("Error parsing logs for pod %s: %v", podName, err)
			continue
		}
		logger.Infof("Fetched %d new parsed documents from logs for pod %s.", len(parsedDocs), podName)

		// Continue with storage/merging and then Parquet writing...
		// (Assume you call your storage and parquet packages as before, now with ParsedDocument)
	}
}
