/*
s3report-go is a s3 bucket report generator.
It uses default AWS credentials to authenticate with AWS API's and establish a client.
Once the script authenticates it lists all buckets, and gathers information about each bucket.


Usage:

    s3report-go [flags] [path ...]

The flags are:

    -f
        The filename of the generated csv file.

*/
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
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
	encStatus  string
	encType    string
	logStatus  string
	logBucket  string
	polStatus  bool
}

// Query the s3 bucket and returns the s3 versioning status.
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

// Query the s3 bucket and returns the s3 encryption status and type.
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

// Query the s3 bucket and returns the s3 logging status and logging bucket(if applicable).
func getLogging(n string, c aws.Config, r string) (l string, b string) {
	s3Client := s3.NewFromConfig(c, func(o *s3.Options) {
		o.Region = r
	})
	resp, err := s3Client.GetBucketLogging(context.TODO(), &s3.GetBucketLoggingInput{Bucket: &n})
	if err != nil {
		log.Fatalf("failed to get bucket logging status, %v", err)
	}
	if resp.LoggingEnabled != nil {
		return "Enabled", *resp.LoggingEnabled.TargetBucket
	} else {
		return "Not Enabled", "None"
	}
}

// Query the s3 bucket and returns whether the bucket is public facing or not.
func isPublic(n string, c aws.Config, r string) (p bool) {
	s3Client := s3.NewFromConfig(c, func(o *s3.Options) {
		o.Region = r
	})
	resp, err := s3Client.GetBucketPolicyStatus(context.TODO(), &s3.GetBucketPolicyStatusInput{Bucket: &n})
	if err != nil {
		return false
	}
	return resp.PolicyStatus.IsPublic
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
		lStatus, lBucket := getLogging(*bucket.Name, cfg, bLocation)
		pStatus := isPublic(*bucket.Name, cfg, bLocation)
		bucketData = append(bucketData, &s3Bucket{*bucket.Name, bLocation, vStatus, eStatus, eType, lStatus, lBucket, pStatus})
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
	data = append(data, []string{"Name", "Region", "Versioning", "Encryption Status", "Encryption Type", "Logging", "Logging Bucket", "Public"})
	for _, record := range bucketData {
		row := []string{record.name, record.region, record.versioning, record.encStatus, record.encType, record.logStatus, record.logBucket, strconv.FormatBool(record.polStatus)}
		data = append(data, row)
	}
	errData := w.WriteAll(data)
	if errData != nil {
		return
	}
}
