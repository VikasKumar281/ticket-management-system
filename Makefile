.PHONY: run build test fmt vet docker-build docker-run

run:
	go run .

build:
	go build -o bin/ticket-system .

test:
	go test ./... -v

fmt:
	gofmt -w .

vet:
	go vet ./...

docker-build:
	docker build -t ticket-system .

docker-run:
	docker run -p 8080:8080 --env-file .env ticket-system
