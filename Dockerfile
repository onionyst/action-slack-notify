# syntax=docker/dockerfile:1

FROM golang:1.25-alpine3.23 AS build
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download

COPY main.go ./

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN --mount=type=cache,id=gobuild,target=/root/.cache/go-build. \
    go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/slack-notify ./main.go

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/slack-notify /slack-notify

USER 65532:65532

ENTRYPOINT ["/slack-notify"]
