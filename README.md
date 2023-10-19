# go-s3-upload

Simple S3 upload CLI written in Go.

This package does one thing, upload a file to S3 using the AWS V2 API.
It's useful for CI/CD environments where you don't want to pull in the entire python runtime to use the AWS CLI.

```
S3 Upload

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

```

The `-e` flag is useful if you want to test this out on something like [LocalStack](https://localstack.cloud/)
