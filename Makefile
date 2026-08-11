SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

export GOPROXY := https://proxy.golang.org,direct

BINARY_NAME := repo-opener
GOBIN       := $(shell go env GOPATH)/bin
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GOLDFLAGS   := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE) -X main.BuiltBy=makefile

# Аргументы для run-cmd / run-bin: make run-cmd ARGS="-version"
ARGS ?=

# Проверка, что внешний инструмент установлен: $(call need,trivy,https://...)
define need
	@command -v $(1) >/dev/null 2>&1 || { echo "$(1) is not installed: $(2)" >&2; exit 1; }
endef

.PHONY: help
help: ## Показать список доступных целей
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## --- Запуск и сборка ---------------------------------------------------

.PHONY: run-cmd
run-cmd: ## Запустить через go run
	go run . $(ARGS)

.PHONY: run-bin
run-bin: build-bin ## Запустить собранный бинарник
	$(GOBIN)/$(BINARY_NAME) $(ARGS)

.PHONY: tidy
tidy: ## Привести в порядок go-модули
	go mod tidy

.PHONY: build-bin
build-bin: ## Собрать бинарник в $(GOBIN)
	go mod download
	CGO_ENABLED=0 go build -ldflags '$(GOLDFLAGS)' -o $(GOBIN)/$(BINARY_NAME) .

.PHONY: install
install: ## Собрать и установить локально с информацией о версии
	CGO_ENABLED=0 go install -ldflags '$(GOLDFLAGS)' .

## --- Форматирование и статический анализ -------------------------------

.PHONY: fmt
fmt: ## Прогнать форматтеры
	gofmt -s -w .
	goimports -format-only -d -l -v -w .

.PHONY: vet
vet: ## Прогнать go vet
	go vet ./...

## --- Тесты -------------------------------------------------------------

.PHONY: test
test: ## Все тесты: покрытие, race detector, отчёт
	go test -coverprofile=cover.out -v ./...
	$(MAKE) test-race
	$(MAKE) test-coverage

.PHONY: test-short
test-short: ## Короткие тесты
	go test --short -coverprofile=cover.out -v ./...

.PHONY: test-coverage
test-coverage: ## Показать отчёт о покрытии
	go tool cover -func=cover.out

.PHONY: test-race
test-race: ## Тесты с race detector
	go test -race -v ./...

.PHONY: test-watch
test-watch: ## Перезапускать тесты при изменении файлов (watchexec)
	$(call need,watchexec,https://github.com/watchexec/watchexec#installation)
	watchexec -c clear -o do-nothing -d 100ms --exts go 'pkg=".$${WATCHEXEC_COMMON_PATH/$$PWD/}/..."; echo "running tests for $$pkg"; go test "$$pkg"'

## --- Линтеры и безопасность --------------------------------------------

.PHONY: lint
lint: lint-golangci lint-bearer lint-trivy lint-govulncheck lint-zizmor ## Прогнать все линтеры и сканеры

.PHONY: lint-golangci
lint-golangci: ## Прогнать golangci-lint
	$(call need,golangci-lint,https://golangci-lint.run/welcome/install/)
	golangci-lint -v run --output.text.colors

.PHONY: lint-bearer
lint-bearer: ## Прогнать Bearer security scan
	$(call need,bearer,https://docs.bearer.com/quickstart/)
	bearer scan . --config-file bearer.yml

.PHONY: lint-trivy
lint-trivy: ## Прогнать Trivy filesystem scan (vuln, secret, misconfig)
	$(call need,trivy,https://trivy.dev/latest/getting-started/installation/)
	trivy fs --config trivy.yaml .

.PHONY: lint-govulncheck
lint-govulncheck: ## Проверить зависимости на известные уязвимости
	$(call need,govulncheck,https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
	govulncheck ./...

.PHONY: lint-zizmor
lint-zizmor: ## Прогнать аудит безопасности GitHub Actions
	$(call need,zizmor,https://docs.zizmor.sh/installation/)
	# Аргумент "." — не только workflow'ы: в набор по умолчанию входят ещё
	# dependabot.yaml и composite actions. Должно совпадать с тем, что
	# передаёт zizmor-action в CI, иначе локальный прогон врёт.
	zizmor .

## --- Релиз -------------------------------------------------------------

.PHONY: release-check
release-check: ## Проверить конфигурацию GoReleaser
	$(call need,goreleaser,https://goreleaser.com/install/)
	goreleaser check

.PHONY: release-snapshot
release-snapshot: ## Собрать локальный snapshot-релиз без публикации
	$(call need,goreleaser,https://goreleaser.com/install/)
	goreleaser release --snapshot --clean --skip=publish

## --- Прочее ------------------------------------------------------------

.PHONY: clean
clean: ## Удалить артефакты сборки и тестов
	rm -rf dist cover.out
