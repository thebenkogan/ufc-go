build:
	go build -o bin/main cmd/main.go

run: build
	./bin/main

test:
	go test --race -v -count=1 ./...