ARG BUILD_HASH="0000000000000000000000000000000000000000"
ARG BUILD_ID="r0"
ARG APP_VERSION="000000-r0"

FROM golang:1.26 AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0



FROM base AS test

RUN --mount=type=cache,target=/root/.cache/go-build \
	go test -v -trimpath ./...



FROM base AS builder

ARG BUILD_HASH
ARG BUILD_ID
ARG APP_VERSION
RUN BUILD_HASH=${BUILD_HASH} \
	BUILD_ID=${BUILD_ID} \
	APP_VERSION=${APP_VERSION} \
	./scripts/gen-version-file.sh

ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
	mkdir /dist \
	&& GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /dist/arm64 . \
	&& GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /dist/amd64 . \
	&& "/dist/${TARGETARCH}" version

FROM alpine:3.21 AS certs
RUN apk add --no-cache ca-certificates

FROM scratch AS build
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /
COPY --from=builder /dist/ /



FROM scratch AS app
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ARG TARGETARCH
COPY "${TARGETARCH}" /oras-get

USER 65532:65532
ENTRYPOINT ["/oras-get"]
CMD ["--help"]
