FROM golang:alpine as build

WORKDIR /app
COPY . .

RUN apk --no-cache add ca-certificates git openssh make

RUN go build -ldflags="-w -s -extldflags '-static' -X 'main.Version=$(git rev-parse HEAD)'" -o /usr/local/bin/s3upload .

FROM alpine:latest as final-base

RUN apk --no-cache add curl jq sqlite-libs git ca-certificates tzdata

ENV USER=uploader
ENV UID=12345
ENV GID=23456

WORKDIR /app

RUN addgroup uploader && adduser \
    --disabled-password \
    --gecos "" \
    --home "$(pwd)" \
    --ingroup "$USER" \
    --uid "$UID" \
    "$USER" && \
	chown -R uploader:uploader /usr/local/bin/ && \
	chown -R uploader:uploader /app

LABEL org.opencontainers.image.title="Go S3 Uploader"
LABEL org.opencontainers.image.description="A simple CLI for uploading files to S3"
LABEL org.opencontainers.image.vendor="Bacchus Jackson"
LABEL org.opencontainers.image.licenses="GNUPL"
LABEL io.artifacthub.package.readme-url="https://raw.githubusercontent.com/BacchusJackson/go-s3-upload/main/README.md"
LABEL io.artifacthub.package.license="GNUPL"

# Final image in a CI environment, assumes binaries are located in ./bin
# This is for pulling in prebuilt binaries and doesn't depend on the build job
FROM final-base as final-ci

COPY ./bin/s3upload /usr/local/bin/s3upload

USER uploader

# Final image if building locally and build dependencies are needed
FROM final-base

COPY --from=build /usr/local/bin/s3upload /usr/local/bin/s3upload

ENTRYPOINT [ "/usr/local/bin/s3upload" ]
