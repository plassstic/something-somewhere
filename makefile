include .env

.PHONY: up uprebuild down api_uprebuild run_migrations validate_sqlc generate_sqlc

up:
	docker compose -f docker-compose.yml up -d

uprebuild:
	docker compose -f docker-compose.yml up -d --build

down:
	docker compose -f docker-compose.yml up -d --build

api_uprebuild:
	docker compose -f docker-compose.yml up 'api' -d --build

run_migrations:
	docker compose -f docker-compose.yml up 'migrate' -d

validate_sqlc:
	sqlc compile

generate_sqlc:
	sqlc generate
