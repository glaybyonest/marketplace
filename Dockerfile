# syntax=docker/dockerfile:1.7

FROM golang:1.24.1-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/api /app/api
COPY --from=builder /out/migrate /app/migrate
EXPOSE 8080
ENTRYPOINT ["/app/api"]
