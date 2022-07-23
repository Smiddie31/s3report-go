package main

import (
	"context"
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Bucket struct {
	name       string
	region     string
	versioning string
	logging    bool
	encStatus  string
	encType    string
}

func getVersioning(n string, c aws.Config, r string) (v string) {
	s3Client := s3.NewFromConfig(c, func(o *s3.Options) {
		o.Region = r
	})
	ver, err := s3Client.GetBucketVersioning(context.TODO(), &s3.GetBucketVersioningInput{Bucket: &n})
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

func getEncryption(n string, c aws.Config, r string) (v string, t string) {
	s3Client := s3.NewFromConfig(c, func(o *s3.Options) {
		o.Region = r
	})
	resp, err := s3Client.GetBucketEncryption(context.TODO(), &s3.GetBucketEncryptionInput{Bucket: &n})
	if err != nil {
		return "Not Enabled", "None"
	}
	switch resp.ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm {
	case "AES256":
		return "Enabled", "SSE"
	case "aws:kms":
		return "Enabled", "KMS"
	default:
		return "Not Enabled", "None"
	}
}

func main() {
	var bucketData []*s3Bucket
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {})
	buckets, awsErr := s3Client.ListBuckets(context.TODO(), nil)
	if awsErr != nil {
		log.Fatalf("Couldn't list buckets: %v", err)
		return
	}

	for _, bucket := range buckets.Buckets {
		bL, blErr := s3Client.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: bucket.Name})
		if blErr != nil {
			log.Fatalf("Couldn't locate bucket: %v", blErr)
		}
		bLocation := string(bL.LocationConstraint)
		if bLocation == "" {
			bLocation = "us-east-1"
		}
		vStatus := getVersioning(*bucket.Name, cfg, bLocation)
		eStatus, eType := getEncryption(*bucket.Name, cfg, bLocation)
		bucketData = append(bucketData, &s3Bucket{*bucket.Name, bLocation, vStatus, false, eStatus, eType})
	}
	file, err := os.Create("bucket-data.csv")
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
	data = append(data, []string{"Name", "Region", "Versioning", "Logging", "Encryption Status", "Encryption Type"})
	for _, record := range bucketData {
		row := []string{record.name, record.region, record.versioning, strconv.FormatBool(record.logging), record.encStatus, record.encType}
		data = append(data, row)
	}
	errData := w.WriteAll(data)
	if errData != nil {
		return
	}
}
