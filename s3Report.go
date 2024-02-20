/*
s3report-go is a s3 bucket report generator.
It uses default AWS credentials to authenticate with AWS API's and establish a client.
Once the script authenticates it lists all buckets, and gathers information about each bucket.

Usage:

	s3report-go [flags] [path ...]

The flags are:

	-f
	    The filename of the generated csv file.
*/package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Smiddie31/s3Tools"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Bucket struct {
	name       string
	region     string
	versioning string
	encStatus  string
	encType    string
	logStatus  string
	logBucket  string
	polStatus  bool
}

func main() {
	var fName string
	flag.StringVar(&fName, "f", "bucket-data", "Specify filename. Default is 'bucket-data'")
	flag.Parse()
	var bucketData []*s3Bucket
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {})
	buckets, awsErr := s3Tools.ListBuckets(context.Background(), s3Client)
	if awsErr != nil {
		log.Fatalf("Couldn't list buckets: %v", err)
		return
	}

	for _, bucket := range buckets.Buckets {
		bL, blErr := s3Tools.GetBucketLocation(context.Background(), s3Client, *bucket.Name)
		if blErr != nil {
			log.Fatalf("Couldn't locate bucket: %v", blErr)
		}
		bLocation := string(bL.LocationConstraint)
		if bLocation == "" {
			bLocation = "us-east-1"
		}
		s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.Region = bLocation
		})
		vStatus := s3Tools.GetBucketVersioning(context.Background(), s3Client, *bucket.Name)
		eStatus, eType := s3Tools.GetBucketEncryption(context.Background(), s3Client, *bucket.Name)
		lStatus, lBucket := s3Tools.GetBucketLogging(context.Background(), s3Client, *bucket.Name)
		pStatus := s3Tools.GetBucketPolicyStatus(context.Background(), s3Client, *bucket.Name)
		bucketData = append(
			bucketData,
			&s3Bucket{*bucket.Name, bLocation, vStatus, eStatus, eType, lStatus, lBucket, pStatus},
		)
	}
	s := fmt.Sprintf("%v.csv", fName)

	file, err := os.Create(s)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalln("failed to close file", err)
		}
	}(file)
	if err != nil {
		log.Fatalln("failed to open file", err)
	}
	w := csv.NewWriter(file)
	defer w.Flush()
	var data [][]string
	data = append(
		data,
		[]string{
			"Name",
			"Region",
			"Versioning",
			"Encryption Status",
			"Encryption Type",
			"Logging",
			"Logging Bucket",
			"Public",
		},
	)
	for _, record := range bucketData {
		row := []string{
			record.name,
			record.region,
			record.versioning,
			record.encStatus,
			record.encType,
			record.logStatus,
			record.logBucket,
			strconv.FormatBool(record.polStatus),
		}
		data = append(data, row)
	}
	errData := w.WriteAll(data)
	if errData != nil {
		return
	}
}
