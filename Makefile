include .env
export

APP_NAME=vein
CMD_PATH=./cmd/api


run:
	go run $(CMD_PATH)

seed:
	go run ./cmd/seed

build:
	go build -o bin/$(APP_NAME) $(CMD_PATH)

build-cli:
	go build -o bin/veincli ./cmd/veincli

test:
	go test ./...

test-integration:
	INTEGRATION_TEST_DSN="$(DB_DSN)" go test ./... -run TestE2E -v

fmt-check:
	./scripts/check_fmt.sh

migration-check:
	./scripts/check_migrations.sh

vet:
	go vet ./...

lint:
	$(shell go env GOPATH)/bin/golangci-lint run ./...

security-check:
	gosec ./...

vuln-check:
	govulncheck ./...

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
