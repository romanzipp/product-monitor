.PHONY: build run tidy vet deploy clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/portasplit-monitor ./cmd/portasplit-monitor

run:
	go run ./cmd/portasplit-monitor

tidy:
	go mod tidy

vet:
	go vet ./...

deploy:
	@bash deploy/deploy.sh

clean:
	rm -rf bin
