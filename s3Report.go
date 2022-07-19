package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Bucket struct {
	name       string
	versioning string
	logging    bool
}

func getVersioning(n string, c *s3.Client) (v string) {
	ver, err := c.GetBucketVersioning(context.TODO(), &s3.GetBucketVersioningInput{Bucket: &n})
	if err != nil {
		log.Fatalf("failed to get bucket versioning status, %v", err)
	}
	switch ver.Status {
	case "Enabled":
		return "Enabled"
	case "Suspended":
		return "Suspended"
	default:
		return "Not Enabled"
	}
}

func main() {
	var bucketData []*s3Bucket
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-southeast-2"))
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg)
	buckets, awserr := s3Client.ListBuckets(context.TODO(), nil)
	if awserr != nil {
		fmt.Printf("Couldn't list buckets: %v", err)
		return
	}

	for _, bucket := range buckets.Buckets {
		vStatus := getVersioning(*bucket.Name, s3Client)
		bucketData = append(bucketData, &s3Bucket{*bucket.Name, vStatus, false})
	}
	file, err := os.Create("bucketdata.csv")
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)
	if err != nil {
		log.Fatalln("failed to open file", err)
	}
	w := csv.NewWriter(file)
	defer w.Flush()
	// Using WriteAll
	var data [][]string
	data = append(data, []string{"Name", "Versioning", "Logging"})
	for _, record := range bucketData {
		row := []string{record.name, record.versioning, strconv.FormatBool(record.logging)}
		data = append(data, row)
	}
	errData := w.WriteAll(data)
	if errData != nil {
		return
	}
}
