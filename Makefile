# --- CONFIG ---
REGISTRY=84.247.175.107:5000
IMAGE=$(REGISTRY)/szaszki-backend-go
TAG=latest

# --- COMMANDS ---

# Build Docker image locally
build:
	docker build -t $(IMAGE):$(TAG) .

# Push image to private registry
push:
	docker push $(IMAGE):$(TAG)

# Build and push in one step (recommended)
publish:
	docker buildx build -t $(IMAGE):$(TAG) --push .

# Optional: tag with git commit hash
publish-hash:
	docker buildx build -t $(IMAGE):$(shell git rev-parse --short HEAD) --push .

# Optional: tag with date (useful for tracking versions)
publish-date:
	docker buildx build -t $(IMAGE):$(shell date +%Y%m%d-%H%M%S) --push .


# --- SCRIPTS ---

run:
	go run .\cmd\server\main.go

# --- TESTING ---

# Run all tests with pretty output
test:
	gotestsum --format testname ./internal/chessengine

bench:
	go test ./internal/chessengine -run=^$$ -bench=. -benchmem > bench.txt
	benchstat bench.txt bench.txt