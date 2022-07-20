package main

import (
	"context"
	"encoding/csv"
	"github.com/aws/aws-sdk-go-v2/aws"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Bucket struct {
	name       string
	region     string
	versioning string
	logging    bool
}

func getRegion(b *s3.GetBucketLocationOutput) (s string) {
	switch b.LocationConstraint {
	case "af-south-1":
		s = "af-south-1"
	case "ap-east-1":
		s = "ap-east-1"
	case "ap-northeast-1":
		s = "ap-northeast-1"
	case "ap-northeast-2":
		s = "ap-northeast-2"
	case "ap-northeast-3":
		s = "ap-northeast-3"
	case "ap-south-1":
		s = "ap-south-1"
	case "ap-southeast-1":
		s = "ap-southeast-1"
	case "ap-southeast-2":
		s = "ap-southeast-2"
	case "ca-central-1":
		s = "ca-central-1"
	case "cn-north-1":
		s = "cn-north-1"
	case "cn-northwest-1":
		s = "cn-northwest-1"
	case "EU":
		s = "EU"
	case "eu-central-1":
		s = "eu-central-1"
	case "eu-north-1":
		s = "eu-north-1"
	case "eu-south-1":
		s = "eu-south-1"
	case "eu-west-1":
		s = "eu-west-1"
	case "eu-west-2":
		s = "eu-west-2"
	case "eu-west-3":
		s = "eu-west-3"
	case "me-south-1":
		s = "me-south-1"
	case "sa-east-1":
		s = "sa-east-1"
	case "us-east-2":
		s = "us-east-2"
	case "us-gov-east-1":
		s = "us-gov-east-1"
	case "us-gov-west-1":
		s = "us-gov-west-1"
	case "us-west-1":
		s = "us-west-1"
	case "us-west-2":
		s = "us-west-2"
	case "":
		s = "us-east-1"
	}
	return s
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

func main() {
	var bucketData []*s3Bucket
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {})
	buckets, awserr := s3Client.ListBuckets(context.TODO(), nil)
	if awserr != nil {
		log.Fatalf("Couldn't list buckets: %v", err)
		return
	}

	for _, bucket := range buckets.Buckets {
		bL, blerr := s3Client.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: bucket.Name})
		if blerr != nil {
			log.Fatalf("Couldn't locate bucket: %v", blerr)
		}
		bLocation := getRegion(bL)
		vStatus := getVersioning(*bucket.Name, cfg, bLocation)
		bucketData = append(bucketData, &s3Bucket{*bucket.Name, bLocation, vStatus, false})
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
	data = append(data, []string{"Name", "Region", "Versioning", "Logging"})
	for _, record := range bucketData {
		row := []string{record.name, record.region, record.versioning, strconv.FormatBool(record.logging)}
		data = append(data, row)
	}
	errData := w.WriteAll(data)
	if errData != nil {
		return
	}
}
