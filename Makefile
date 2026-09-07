tidy ::
	@go mod tidy && go mod vendor

seed ::
	@go run cmd/seed/main.go

run ::
	@go run cmd/server/main.go

test ::
	@go test -v -count=1 -race ./... -coverprofile=coverage.out -covermode=atomic

docker-up ::
	docker compose up -d

docker-down ::
	docker compose down
