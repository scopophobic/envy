FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /envo-server ./cmd/server

# ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /envo-server /usr/local/bin/envo-server

EXPOSE 8080
ENTRYPOINT ["envo-server"]
