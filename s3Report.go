package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"os"
	"strconv"
)

type s3Bucket struct {
	name       string
	versioning bool
	logging    bool
}

func ListBuckets(client *s3.Client) (*s3.ListBucketsOutput, error) {
	res, err := client.ListBuckets(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func main() {
	var bucketData []*s3Bucket
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}
	s3client := s3.NewFromConfig(cfg)
	buckets, awserr := ListBuckets(s3client)
	if awserr != nil {
		fmt.Printf("Couldn't list buckets: %v", err)
		return
	}

	for _, bucket := range buckets.Buckets {
		bucketData = append(bucketData, &s3Bucket{*bucket.Name, true, false})
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
		row := []string{record.name, strconv.FormatBool(record.versioning), strconv.FormatBool(record.logging)}
		data = append(data, row)
	}
	errData := w.WriteAll(data)
	if errData != nil {
		return
	}
}
