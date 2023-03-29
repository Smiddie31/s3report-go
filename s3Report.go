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

// BucketVersioning is an interface for the AWS API Call GetBucketVersioning
type BucketVersioning interface {
	GetBucketVersioning(ctx context.Context, input *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error)
}

// BucketEncryption is an interface for the AWS API Call GetBucketEncryption
type BucketEncryption interface {
	GetBucketEncryption(ctx context.Context, input *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error)
}

// BucketLogging is an interface for the AWS API Call GetBucketLogging
type BucketLogging interface {
	GetBucketLogging(ctx context.Context, input *s3.GetBucketLoggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketLoggingOutput, error)
}

// BucketVisibility is an interface for the AWS API Call GetBucketPolicy
type BucketVisibility interface {
	GetBucketPolicyStatus(ctx context.Context, input *s3.GetBucketPolicyStatusInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyStatusOutput, error)
}

// BucketListing is an interface for the AWS API Call List Buckets
type BucketListing interface {
	ListBuckets(ctx context.Context, input *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
}

// ListBuckets is a function to list all S3 Buckets
func ListBuckets(ctx context.Context, client BucketListing) (*s3.ListBucketsOutput, error) {
	input := &s3.ListBucketsInput{}
	return client.ListBuckets(ctx, input)
}

// GetBucketVersioning is a function in which gathers the version of a S3 Bucket
func GetBucketVersioning(ctx context.Context, client BucketVersioning, bucketName string) string {
	input := &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	}
	ver, err := client.GetBucketVersioning(ctx, input)
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

// GetBucketEncryption is a function in which gathers the encryption and encryption type of a S3 Bucket
func GetBucketEncryption(ctx context.Context, client BucketEncryption, bucketName string) (string, string) {
	input := &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	}
	enc, err := client.GetBucketEncryption(ctx, input)
	if err != nil {
		return "Not Enabled", "None"
	}
	switch enc.ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm {
	case "AES256":
		return "Enabled", "SSE"
	case "aws:kms":
		return "Enabled", "KMS"
	default:
		return "Not Enabled", "None"
	}

}

// GetBucketLogging is a function in which gathers the logging of a S3 Bucket
func GetBucketLogging(ctx context.Context, client BucketLogging, bucketName string) (string, string) {
	input := &s3.GetBucketLoggingInput{
		Bucket: aws.String(bucketName),
	}
	logr, err := client.GetBucketLogging(ctx, input)
	if err != nil {
		log.Fatalf("failed to get bucket logging status, %v", err)
	}
	if logr.LoggingEnabled != nil {
		return "Enabled", *logr.LoggingEnabled.TargetBucket
	}
	return "Not Enabled", "None"
}

// GetBucketPolicyStatus is a function that determines whether a S3 Bucket is public or not.
func GetBucketPolicyStatus(ctx context.Context, client BucketVisibility, bucketName string) bool {
	input := &s3.GetBucketPolicyStatusInput{
		Bucket: aws.String(bucketName),
	}
	pol, err := client.GetBucketPolicyStatus(ctx, input)
	if err != nil {
		return false
	}
	return pol.PolicyStatus.IsPublic
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
	buckets, awsErr := ListBuckets(context.Background(), s3Client)
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
		s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.Region = bLocation
		})
		vStatus := GetBucketVersioning(context.Background(), s3Client, *bucket.Name)
		eStatus, eType := GetBucketEncryption(context.Background(), s3Client, *bucket.Name)
		lStatus, lBucket := GetBucketLogging(context.Background(), s3Client, *bucket.Name)
		pStatus := GetBucketPolicyStatus(context.Background(), s3Client, *bucket.Name)
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
