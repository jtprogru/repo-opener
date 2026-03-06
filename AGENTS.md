# AGENTS.md — Guidelines for AI Agents

## Project Overview

**repo-opener** — простая утилита на Go для быстрого открытия текущего Git-репозитория в браузере.

- **Модуль:** `github.com/jtprogru/repo-opener`
- **Версия Go:** 1.25.0
- **Основные зависимости:**
  - `github.com/go-git/go-git/v5` — работа с Git
  - `github.com/pkg/browser` — открытие URL в браузере
  - `github.com/stretchr/testify` — тестирование

## Структура проекта

```
repo-opener/
├── main.go           # Основная логика приложения
├── main_test.go      # Юнит-тесты
├── go.mod            # Зависимости
├── Taskfile.yml      # Задачи сборки и тестирования
├── .golangci.yaml    # Конфигурация golangci-lint
└── .goreleaser.yaml  # Конфигурация релизов
```

## Инструменты разработки

### Сборка и запуск

```bash
# Сборка бинарного файла
task build:bin

# Запуск из исходников
task run:cmd

# Запуск собранного бинарника
task run:bin

# Установка локально
task install
```

### Тестирование

```bash
# Все тесты с coverage
task test

# Короткие тесты
task test:short

# Тесты с race detector
task test:race

# Покрытие
task test:coverage
```

### Линтинг

```bash
# Все линтеры
task lint

# golangci-lint
task lint:golangci

# Bearer Security
task lint:bearer
```

### Форматирование

```bash
task fmt
```

## Стиль кода

### Общие правила

- **Идентификаторы:** `camelCase` для переменных, `PascalCase` для экспортируемых
- **Обработка ошибок:** явная, с обёрткой через `fmt.Errorf` с `%w`
- **Константы ошибок:** вынесены в `const` с префиксом `Err`
- **Комментарии:** только для сложной логики, без избыточности

### Специфичные требования

1. **golangci-lint:** строгая конфигурация с множеством включённых линтеров
2. **gochecknoglobals:** глобальные переменные требуют комментария `//nolint:gochecknoglobals`
3. **revive:** максимальная строгость, многие правила отключены явно
4. **nakedret:** `max-func-lines: 1` — запрещены naked returns в функциях > 1 строки
5. **tagliatelle:** JSON-теги в `snake_case`

### Примеры

```go
// Правильно:
const ErrOriginRemoteNotFound = "origin remote not found"

var versionFlag bool //nolint:gochecknoglobals // This is normal

func parseRemoteURL(remoteURL string) (string, error) {
    if u, err := url.Parse(remoteURL); err == nil && u.Scheme != "" {
        return parseStructuredURL(u)
    }
    // ...
}

// Неправильно:
func parseRemoteURL(remoteURL string) (string, error) {
    if err != nil {
        return "", err  // naked return запрещён
    }
}
```

## Тестирование

### Требования к тестам

- ** testify/assert** и **testify/require** для ассертов
- `t.Helper()` для вспомогательных функций
- `t.Cleanup()` для очистки
- Table-driven тесты для множественных кейсов
- `t.Parallel()` для параллельных тестов

### Паттерны

```go
func TestFunction(t *testing.T) {
    t.Parallel()

    t.Run("subtest name", func(t *testing.T) {
        t.Parallel()
        // тест
    })
}

// Вспомогательная функция
func initTempRepo(t *testing.T) string {
    t.Helper()
    // ...
}
```

## CI/CD

Проект использует GitHub Actions:

- **lint.yaml** — golangci-lint
- **tests.yaml** — тесты
- **goreleaser.yaml** — релизы
- **bearer.yaml** — security scan

## Версионирование

Переменные сборки передаются через ldflags:

```bash
-X main.Version=<version>
-X main.Commit=<commit>
-X main.Date=<date>
-X main.BuiltBy=<builder>
```

## Ключевые функции

| Функция | Описание |
|---------|----------|
| `main()` | Точка входа, парсинг флагов, оркестрация |
| `checkGitRepo()` | Проверка наличия Git-репозитория |
| `getRemoteURL()` | Получение URL remote "origin" |
| `parseRemoteURL()` | Парсинг URL (HTTP/SSH) |
| `buildWebURL()` | Конвертация в веб-URL |

## Важные замечания

1. **CGO_ENABLED=0** для статической сборки
2. **GOPROXY:** `https://proxy.golang.org,direct`
3. Все команды через **Taskfile.yml**, не используйте `go build` напрямую
4. Перед коммитом: `task lint && task test`
