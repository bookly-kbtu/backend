-include .env
export

.PHONY: run build test lint swagger import-sources import-cities import import-runs migration-up migration-down migration-status migration-reset migration-redo migration-create

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api
	go build -o bin/migrate ./cmd/migrate
	go build -o bin/command ./cmd/command

test:
	go test ./...

lint:
	go vet ./...

SWAG = go run github.com/swaggo/swag/cmd/swag@v1.16.6

# Generates docs/swagger.json (not in git). Run after changing handlers or DTOs.
swagger:
	$(SWAG) fmt
	$(SWAG) init -g cmd/api/main.go -o docs --parseInternal --outputTypes json

migration-up:
	go run ./cmd/migrate up

migration-down:
	go run ./cmd/migrate down

migration-status:
	go run ./cmd/migrate status

migration-reset:
	go run ./cmd/migrate reset

migration-redo:
	go run ./cmd/migrate redo

# make migration-create name=add_reviews
migration-create:
	go run ./cmd/migrate create $(name) sql

# make import source=zapis_kz city=1 max=10
import-sources:
	go run ./cmd/command sources

import-cities:
	go run ./cmd/command cities -source $(or $(source),zapis_kz)

import:
	go run ./cmd/command import -source $(or $(source),zapis_kz) -city $(city) $(if $(max),-max-firms $(max)) $(if $(snapshots),-snapshots)

import-runs:
	go run ./cmd/command runs
