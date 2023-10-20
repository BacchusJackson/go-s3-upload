package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	"github.com/lmittmann/tint"
)

const Version = "v1.0.1"

const help = `S3 Upload

Upload a file to S3 using the AWS SDK.

usage:
	s3upload [OPTIONS] -f TARGET_FILE -k S3_KEY -b BUCKET

Required Flags:
	-f file Target file to upload
	-k key Target key for the file in S3
	-b bucket Target bucket for the upload

Optional Flags:
	-h help Message and command usage
	-q quiet Supress logging output

Additional Flags:
	-e S3 BaseEndpoint Used for debugging purposes
	
Example:

	AWS_DEFAULT_PROFILE="my-profile" s3upload -f file-1.txt -k path/to/file/file-1.txt -b my-bucket
`

const ExitOK = 0
const ExitSystemFailure = 1
const ExitAPIFailure = 2

func main() {
	fileFlag := flag.String("f", "", "Target file to upload")
	keyFlag := flag.String("k", "", "Target Key for the file in S3")
	bucketFlag := flag.String("b", "", "Target bucket for the upload")
	endpoint := flag.String("e", "", "BaseEndpoint URL, used for debugging")

	quietFlag := flag.Bool("q", false, "supressing logging output")
	helpFlag := flag.Bool("h", false, "help message and command usage")

	flag.Parse()
	if helpFlag != nil && *helpFlag {
		fmt.Fprintf(os.Stderr, help)
		os.Exit(ExitSystemFailure)
	}

	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{Level: logLevel, TimeFormat: time.TimeOnly}))
	slog.SetDefault(logger)

	if quietFlag != nil && *quietFlag {
		logLevel.Set(slog.LevelWarn)
	}

	stringFlags := map[string]*string{"file": fileFlag, "key": keyFlag, "bucket": bucketFlag}

	// Check to make sure string flags are defined and not blank
	var flagErrs error
	for flagName, flagValue := range stringFlags {
		if flagValue == nil || strings.TrimSpace(*flagValue) == "" {
			err := fmt.Errorf("%-6s flag is required", flagName)
			slog.Error(err.Error())
			flagErrs = errors.Join(flagErrs, err)
		}
	}

	if flagErrs != nil {
		os.Exit(ExitSystemFailure)
	}

	slog.Debug("load AWS shared config profile")

	// Load the AWS Configuration file from Shared Config
	// https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html
	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		slog.Error("failed to load AWS default profile. Use AWS_DEFAULT_PROFILE", "err", err,
			"help_url", "https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html")
	}

	slog.Debug("open", "filename", *fileFlag)
	f, err := os.Open(*fileFlag)
	if err != nil {
		slog.Error("failed open", "filename", *fileFlag, "err", err)
		os.Exit(ExitSystemFailure)
	}
	if endpoint == nil || strings.TrimSpace(*endpoint) == "" {
		endpoint = nil
	}

	err = s3Upload(cfg, *keyFlag, *bucketFlag, endpoint, f)

	if err != nil {
		os.Exit(ExitAPIFailure)
	}

	os.Exit(ExitOK)
}

func s3Upload(cfg aws.Config, key string, bucket string, endpoint *string, r io.Reader) error {

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = endpoint
		o.Logger = logging.LoggerFunc(func(classification logging.Classification, format string, v ...interface{}) {
			if classification == logging.Warn {
				slog.Warn(fmt.Sprintf(format, v))
				return
			}
			slog.Debug(fmt.Sprintf(format, v))
		})
	})

	slog.Debug("init s3 client", "region", cfg.Region, "key", key, "bucket", bucket)

	putReq := &s3.PutObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
		Body:   r,
	}

	slog.Info("execute s3 put")
	_, err := client.PutObject(context.Background(), putReq)

	if err != nil {
		slog.Error("failed s3 put", "err", err)
		return err
	}

	slog.Info("success s3 put")
	return nil
}
