include .env
export

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

migrate-create:
	migrate create -seq -ext=.sql -dir=./migrations $(name)

migrate-up:
	migrate -path=./migrations -database "$(DB_DSN)" up

migrate-down:
	migrate -path=./migrations -database "$(DB_DSN)" down

migrate-force:
	migrate -path=./migrations -database "$(DB_DSN)" force 1