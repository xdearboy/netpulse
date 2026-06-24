.PHONY: build test run docker deploy

build:
	go build -o bin/netpulse ./cmd/server

test:
	go test ./... -count=1 -race

run:
	go run cmd/server/main.go

docker:
	docker build -t netpulse .

deploy:
	kubectl apply -k deploy/

logs:
	kubectl logs -f deployment/netpulse -n netpulse
