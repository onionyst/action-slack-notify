FROM golang:1.18-alpine3.15 AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags "-extldflags '-static'" -o /go/bin/slack-notify main.go

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/slack-notify /slack-notify

ENTRYPOINT [ "/slack-notify" ]
