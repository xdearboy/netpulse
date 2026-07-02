.PHONY: build test run lint fmt docker deploy

build:
	go build -o bin/netpulse ./cmd/server

test:
	go test ./... -count=1 -race

run:
	go run cmd/server/main.go

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	cd frontend && npx prettier --write "src/**/*.{ts,tsx,css}"

docker:
	docker build -t netpulse .

deploy:
	kubectl apply -k deploy/

logs:
	kubectl logs -f deployment/netpulse -n netpulse
