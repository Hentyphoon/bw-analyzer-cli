# Brood War replay analyzer. Targets assume a POSIX shell and Go on PATH.

BIN          ?= bin/bwa
DATABASE_URL ?= postgres://bwa:bwa@localhost:5432/bwa?sslmode=disable
# The race detector needs cgo and a C compiler. Override with RACE= on
# machines without one (for example Windows without gcc).
RACE         ?= -race

export DATABASE_URL

.PHONY: build test vet lint db-up db-down migrate bench clean

build:
	go build -o $(BIN) ./cmd/bwa

test:
	go test $(RACE) ./...

vet:
	go vet ./...

lint: vet
	golangci-lint run ./...

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

migrate: build
	$(BIN) migrate

# Implemented in milestone M4.
bench:
	@echo "make bench: not implemented until milestone M4" >&2
	@exit 1

clean:
	rm -rf bin
