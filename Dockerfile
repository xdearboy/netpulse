FROM --platform=linux/amd64 node:22-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend/ ./
RUN npx vite build

FROM --platform=linux/amd64 golang:1.25-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/dist ./cmd/server/static/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /netpulse ./cmd/server

FROM --platform=linux/amd64 alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
COPY --from=backend /netpulse /netpulse
EXPOSE 8080
ENTRYPOINT ["/netpulse"]
