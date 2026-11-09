.PHONY: tidy test test-unit test-integ build pack pack-linux run-webhttp run-history run-push run-mqtt deps-up smoke

tidy:
	go mod tidy

# 全量：单测 + SQLite 集成测（CI 同款）
test:
	go test ./... -count=1 -timeout 120s

test-unit:
	go test ./backend/pkg/... ./backend/internal/biz/internal/domain/... ./backend/internal/webhttp/middleware/... -count=1

test-integ:
	go test ./backend/internal/biz/service/ -count=1 -timeout 120s

EXE :=
ifeq ($(OS),Windows_NT)
EXE := .exe
endif

build:
	go build -o bin/webhttp$(EXE) ./backend/cmd/webhttp
	go build -o bin/history$(EXE) ./backend/cmd/history
	go build -o bin/push$(EXE) ./backend/cmd/push
	go build -o bin/mqtt$(EXE) ./backend/cmd/mqtt

# 一键发布包（默认 Windows amd64）
pack:
	powershell -File tools/scripts/pack.ps1

pack-linux:
	powershell -File tools/scripts/pack.ps1 -Target linux

run-webhttp:
	go run ./backend/cmd/webhttp

run-history:
	go run ./backend/cmd/history

run-push:
	go run ./backend/cmd/push

run-mqtt:
	go run ./backend/cmd/mqtt

deps-up:
	docker compose -f deploy/compose/docker-compose.yml up -d

smoke:
	powershell -File tools/scripts/smoke.ps1 -AutoStart
