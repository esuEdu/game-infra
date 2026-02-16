.PHONY: docker-up docker-up-build docker-down docker-clean

docker-up:
	docker compose -f docker-compose.local.yml up

docker-up-build:
	docker compose -f docker-compose.local.yml up --build

docker-down:
	docker compose -f docker-compose.local.yml down

docker-clean:
	docker compose -f docker-compose.local.yml down --rmi local
