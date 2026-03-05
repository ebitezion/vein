APP_NAME=vein
CMD_PATH=./cmd/api

run:
	go run $(CMD_PATH)

build:
	go build -o bin/$(APP_NAME) $(CMD_PATH)

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin