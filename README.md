
# Order Processing Service (Digital Goods Store Core)

Сервис обработки заказов цифровых товаров с гарантией доставки exactly-once, обработкой сбоев поставщиков и фоновым восстановлением зависших заказов.

## Быстрый запуск

1. Запуск PostgreSQL и приложения через Docker Compose:
```bash
docker-compose up -d --build

```

2. Автоматический прогон интеграционного теста (Bash + curl):

```bash
chmod +x tests/integration.sh
./tests/integration.sh

```

## Тестирование

### 1. Интеграционные тесты через [gcli](https://github.com/Votline/Gurl-cli)

В каталоге `tests/` подготовлены сценарии:

```bash
gcli run tests/orders_check.gurlf
gcli run tests/webhooks_check.gurlf
gcli run tests/reaper_check.gurlf
gcli run tests/suppliers_check.gurlf

```

### 2. Нагрузочное тестирование гонок (k6)

Проверка `SELECT FOR UPDATE` и идемпотентности при одновременной отправке 50 дубликатов вебхуков:

```bash
k6 run tests/load_test.js

```

## Время разработки

На реализацию ядра, интеграций, фонового воркера и тестов ушло порядка 5 часов.
