APP ?= minisnap
BINARY ?= bin/$(APP)
IMAGE ?= evarle/minisnap
TAG ?= latest

.PHONY: build test lint fmt run docker-build docker-push docker-run clean

build:
	go build -o $(BINARY) ./cmd/server

test:
	go test ./...

lint:
	@echo "Checking gofmt..."
	@fmt_files=$$(gofmt -l .); \
	if [ -n "$$fmt_files" ]; then \
		echo "Files requiring gofmt:"; \
		echo "$$fmt_files"; \
		exit 1; \
	fi
	go vet ./...

fmt:
	gofmt -w .

run:
	go run ./cmd/server

docker-build:
	docker build -t $(IMAGE):$(TAG) .

docker-push: docker-build
	docker push $(IMAGE):$(TAG)

# Run the container locally with the current directory mounted as content storage
# Usage: make docker-run TAG=latest

docker-run:
	docker run --rm -p 8080:8080 --env-file .env -v $(PWD)/content:/app/content $(IMAGE):$(TAG)

clean:
	rm -rf bin build
