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
	-p profile Share Profile, if not defined will default to AWS_PROFILE
	-r region AWS Region, if not defined will default to AWS_REGION

Additional Flags:
	-e S3 BaseEndpoint Used for debugging purposes

Environment Variables:
	AWS_PROFILE optional the profile identifier in the shared config
	AWS_REGION optional the target region for the S3 bucket

Example:

	s3upload -f file-1.txt -k path/to/file/file-1.txt -b my-bucket

Parameters order of precedence is as follows:

1. Flag Options
2. Environment Variables
3. Profile Config Options, if supported
`

const ExitOK = 0
const ExitSystemFailure = 1
const ExitAPIFailure = 2

func main() {
	fileFlag := flag.String("f", "", "Target file to upload")
	keyFlag := flag.String("k", "", "Target Key for the file in S3")
	bucketFlag := flag.String("b", "", "Target bucket for the upload")
	endpointFlag := flag.String("e", "", "BaseEndpoint URL, used for debugging")

	profileFlag := flag.String("p", "", "Shared profile identifier")
	regionFlag := flag.String("r", "", "The target region for the s3 bucket")

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
	profile := os.Getenv("AWS_PROFILE")
	region := os.Getenv("AWS_REGION")

	if profileFlag != nil && strings.TrimSpace(*profileFlag) != "" {
		profile = *profileFlag
	}

	if regionFlag != nil && strings.TrimSpace(*regionFlag) != "" {
		region = *regionFlag
	}

	if flagErrs != nil {
		os.Exit(ExitSystemFailure)
	}

	slog.Debug("load AWS shared config profile")

	// Load the AWS Configuration file from Shared Config
	// https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(profile))

	if err != nil {
		slog.Error("failed to load AWS default profile. Use AWS_PROFILE", "err", err,
			"help_url", "https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html")
	}
	cfg.Region = region

	slog.Debug("config loaded", "region", cfg.Region, "profile", profile)

	slog.Debug("open", "filename", *fileFlag)
	f, err := os.Open(*fileFlag)
	if err != nil {
		slog.Error("failed open", "filename", *fileFlag, "err", err)
		os.Exit(ExitSystemFailure)
	}
	if endpointFlag == nil || strings.TrimSpace(*endpointFlag) == "" {
		endpointFlag = nil
	}

	err = s3Upload(cfg, *keyFlag, *bucketFlag, endpointFlag, f)

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
