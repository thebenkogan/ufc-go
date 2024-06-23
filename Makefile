.PHONY: build run test coverage up down

build:
	go build -o bin/main cmd/main.go

run: build
	./bin/main

test:
	go test -count=1 ./...

coverage:
	mkdir coverage || true
	go test -count=1 -coverpkg=./... -coverprofile ./coverage/cover.out ./...
	go tool cover -html ./coverage/cover.out -o ./coverage/cover.html

up:
	docker compose up -d

down:
	docker compose down