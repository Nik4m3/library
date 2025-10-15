DB_DSN := postgres://library:library@localhost:5432/library?sslmode=disable

pg-up:
	docker compose up -d db

migrate:
	DB_DSN=$(DB_DSN) \
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir ./migrations postgres "${DB_DSN}" up

run-local:
	HTTP_ADDR=:8080 DB_DSN=$(DB_DSN) go run ./cmd/library

up: pg-up migrate
	docker compose up --build api

down:
	docker compose down