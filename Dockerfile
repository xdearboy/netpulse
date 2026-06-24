FROM --platform=linux/amd64 golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /netpulse ./cmd/server

FROM --platform=linux/amd64 alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /netpulse /netpulse
EXPOSE 8080
ENTRYPOINT ["/netpulse"]
