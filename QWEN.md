# QWEN.md — Context for AI Assistant

## Project Overview

**repo-opener** — утилита командной строки на Go для быстрого открытия текущего Git-репозитория в браузере.

- **Модуль:** `github.com/jtprogru/repo-opener`
- **Go версия:** 1.25.0
- **Тип проекта:** CLI-утилита
- **Лицензия:** MIT (см. LICENSE)

## Основные зависимости

```go
github.com/go-git/go-git/v5    // Git-операции
github.com/pkg/browser         // Открытие URL в браузере
github.com/stretchr/testify    // Тестирование
```

## Структура проекта

```
repo-opener/
├── main.go           # Основная логика (main, checkGitRepo, getRemoteURL, parseRemoteURL, buildWebURL)
├── main_test.go      # Юнит-тесты
├── go.mod / go.sum   # Зависимости Go
├── Taskfile.yml      # Задачи сборки, тестирования, линтинга
├── .golangci.yaml    # Конфигурация golangci-lint
├── .goreleaser.yaml  # Конфигурация кросс-платформенных релизов
├── bearer.yml        # Конфигурация security-сканера
└── .github/workflows # CI/CD пайплайны
```

## Сборка и запуск

### Установка зависимостей
```bash
task tidy
```

### Сборка
```bash
# Сборка бинарного файла в $HOME/go/bin/repo-opener
task build:bin

# Локальная установка
task install
```

### Запуск
```bash
# Из исходников
task run:cmd

# Собранным бинарником
task run:bin

# С аргументами
task run:cmd -- -version
```

### Ручные команды (без Taskfile)
```bash
go mod tidy
go build -o repo-opener main.go
./repo-opener
```

## Тестирование

```bash
# Все тесты с coverage и race detector
task test

# Короткие тесты
task test:short

# Race detector
task test:race

# Покрытие
task test:coverage

# Watch mode
task test:watch
```

## Линтинг и форматирование

```bash
# Форматирование кода
task fmt

# Все линтеры (golangci-lint + bearer)
task lint

# Только golangci-lint
task lint:golangci

# Security scan
task lint:bearer

# Go vet
task vet
```

## Стиль кода и соглашения

### Именование
- **Переменные:** `camelCase`
- **Экспортируемые идентификаторы:** `PascalCase`
- **Константы ошибок:** `Err<Description>` (например, `ErrOriginRemoteNotFound`)

### Обработка ошибок
```go
// С обёрткой через fmt.Errorf с %w
return "", fmt.Errorf("failed to open git repository: %w", err)

// Константы ошибок вынесены отдельно
const ErrOriginRemoteNotFound = "origin remote not found"
```

### Глобальные переменные
Требуют комментария `//nolint:gochecknoglobals`:
```go
var versionFlag bool //nolint:gochecknoglobals // This is normal
```

### Требования golangci-lint
- **nakedret:** `max-func-lines: 1` — запрещены naked returns в функциях > 1 строки
- **tagliatelle:** JSON-теги в `snake_case`
- **revive:** максимальная строгость, многие правила настроены индивидуально
- **line-length-limit:** отключён (120 символов не enforced)

## Тестирование — требования

### Библиотеки
- `testify/assert` — для ассертов
- `testify/require` — для обязательных проверок

### Паттерны
```go
// Table-driven тесты
tests := []struct {
    name    string
    host    string
    path    string
    want    string
    wantErr bool
}{...}

// Helper функции
func initTempRepo(t *testing.T) string {
    t.Helper()
    // ...
}

// Parallel тесты
func TestFunction(t *testing.T) {
    t.Parallel()
    t.Run("subtest", func(t *testing.T) {
        t.Parallel()
        // ...
    })
}
```

## CI/CD (GitHub Actions)

- **lint.yaml** — golangci-lint
- **tests.yaml** — тесты
- **goreleaser.yaml** — кросс-платформенные релизы
- **bearer.yaml** — security scan

## Переменные сборки (ldflags)

```bash
-X main.Version=<version>   # Версия (default: "dev")
-X main.Commit=<commit>     # Git commit (default: "none")
-X main.Date=<date>         # Дата сборки (default: "unknown")
-X main.BuiltBy=<builder>   # Кто собрал (default: "unknown")
```

## Ключевые функции

| Функция | Описание |
|---------|----------|
| `main()` | Парсинг флагов, оркестрация |
| `getRemoteURL()` | Получение URL remote "origin" |
| `parseRemoteURL()` | Парсинг URL (HTTP/SSH форматы) |
| `parseStructuredURL()` | Парсинг структурированного URL |
| `parseSSHURL()` | Парсинг SSH-URL (git@host:path) |
| `buildWebURL()` | Конвертация в HTTPS веб-URL |

## Поддерживаемые Git-хосты

### Публичные платформы
- **GitHub:** `github.com`, `www.github.com`
- **GitLab:** `gitlab.com`, `www.gitlab.com`
- **Bitbucket:** `bitbucket.org`, `www.bitbucket.org`

### Приватные инсталляции
- **GitLab CE/EE:** `gitlab.internal.company`, `git.company.com`
- **GitHub Enterprise:** `github.internal.company`
- **Gitea/Forgejo:** `gitea.example.com`, `forgejo.example.com`
- **Другие:** любые кастомные хосты (`git.jtprog.ru`)

**Важно:** Приватные инсталляции **не нормализуются** — хост сохраняется как есть.

## Важные замечания

1. **CGO_ENABLED=0** для статической сборки
2. **GOPROXY:** `https://proxy.golang.org,direct`
3. Перед коммитом: `task lint && task test`
4. Не использовать `go build` напрямую — только через Taskfile
5. Поддерживаемые ОС: linux, windows, darwin, freebsd
6. Архитектуры: amd64, arm64 (darwin 386 игнорируется)

## Команды для разработки (quick reference)

```bash
# Перед коммитом
task lint && task test

# Сборка релиза
task build:bin

# Запуск с флагом
task run:cmd -- -version
```
