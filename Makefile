.PHONY: build test test-race test-integration run lint clean docker-build docker-run

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -v -count=1 -short

test-race:
	go test ./... -v -race -count=1 -short

test-integration:
	go test ./persistence/ -run TestPostgres -v -race -count=1

run: build
	./bin/server

lint:
	go vet ./...

docker-build:
	docker build -t signature-service .

docker-run: docker-build
	docker run --rm -p 8080:8080 signature-service

clean:
	rm -rf bin/
