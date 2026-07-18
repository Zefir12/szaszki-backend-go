# --- CONFIG ---
REGISTRY=registrydedic.zefirlabs.net
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
	gotestsum --format testname ./internal/chessengine ./internal

bench:
	go test ./internal/chessengine -run=^$$ -bench=. -benchmem -count=6 > benchmarkresults/engine-benchmark-v1.0.5.txt
	benchstat benchmarkresults/engine-benchmark-v1.0.4.txt benchmarkresults/engine-benchmark-v1.0.5.txt

bench-full:
	go test ./internal/chessengine -run=^$$ -bench=. -benchmem -count=6 > benchmarkresults/engine-benchmark-full-v1.0.5.txt
	benchstat benchmarkresults/engine-benchmark-full-v1.0.4.txt benchmarkresults/engine-benchmark-full-v1.0.5.txt

benchsession:
	go test ./internal -run=^$$ -bench=. -benchmem -count=6 > benchmarkresults/session-benchmark-v1.0.5.txt
	benchstat benchmarkresults/session-benchmark-v1.0.4.txt benchmarkresults/session-benchmark-v1.0.5.txt

benchsession-full:
	go test ./internal -run=^$$ -bench=. -benchmem -count=6 > benchmarkresults/session-benchmark-full-v1.0.5.txt
	benchstat benchmarkresults/session-benchmark-full-v1.0.4.txt benchmarkresults/session-benchmark-full-v1.0.5.txt