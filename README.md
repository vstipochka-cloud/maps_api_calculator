# API Calculator - Сравнение стоимости геолокационных API сервисов

Полнофункциональное приложение с **отдельными** бэккэнд (Go REST API) и фронтенд (Vanilla JS SPA) компонентами.

## 🚀 Быстрый старт (2 терминала)

### Terminal 1: Backend API (расчеты + логика)
```bash
cd /Users/stipochka/Desktop/api_calculator/calculator_api

go build -o calculator ./cmd/calculator

./calculator
```

**Запустится на:** http://localhost:8080

### Terminal 2: Frontend (веб-интерфейс)
```bash
cd /Users/stipochka/Desktop/api_calculator/calculator_api/frontend

# Option 1: Python (встроенный)
python3 -m http.server 3000

# Option 2: Node.js
npx http-server -p 3000

# Option 3: VS Code Live Server
# Кликни правый на index.html → Open with Live Server
```

**Откроется в браузере:** http://localhost:3000

---

## 📱 Использование

### Веб-интерфейс (пользовательский)
Открыть http://localhost:3000 и:

1. Выбрать нужные API (Geocoding, Routing, etc.)
2. Ввести количество запросов в месяц
3. Нажать "Calculate"
4. Увидеть все варианты с ценами и разбивкой

**Почему 2 сервера?**
- 🎯 Независимое масштабирование
- 🔒 Безопасность (отдельные домены)
- 📦 Dev/Deploy удобство
- 🏗️ Чистая архитектура

**Подробнее:** см. [FRONTEND.md](FRONTEND.md)

### API endpoints (Для интеграции)
Или интегрировать напрямую через REST API (см. ниже)

---

## Архитектура

### 1. GET /health
Проверка статуса сервера.

**Ответ:**
```json
{
  "status": "ok"
}
```

### 2. GET /providers
Получить список доступных провайдеров и поддерживаемые ими API типы.

**Ответ:**
```json
{
  "api_types": [
    "geocoding",
    "routing",
    "map_tiles_raster",
    "map_tiles_vector_2d",
    "map_tiles_vector_3d",
    "static_maps",
    "distance_matrix",
    "elevation"
  ],
  "providers": [
    {
      "id": "google_maps",
      "name": "Google Maps Platform",
      "url": "https://cloud.google.com/maps-platform",
      "apis": ["geocoding", "routing", "map_tiles_raster", ...]
    },
    ...
  ]
}
```

### 3. POST /calculate
Расчет стоимости использования API для заданных параметров.

**Параметры запроса:**
```json
{
  "api_requests": {
    "geocoding": 50000,
    "routing": 10000,
    "map_tiles_raster": 100000,
    "static_maps": 5000
  },
  "disable_new_customer_credit": false,
  "disable_free_tier": false
}
```

**Параметры:**
- `api_requests` *(обязательно)* - объект с количеством запросов для каждого типа API
- `disable_new_customer_credit` *(опционально, по умолчанию false)* - если `true`, не применяется бесплатный кредит для новых клиентов (например, $300 для Google Maps)
- `disable_free_tier` *(опционально, по умолчанию false)* - если `true`, не применяется бесплатный уровень (free tier) для каждого API типа

**Ответ:**
```json
{
  "results": [
    {
      "provider": "openrouteservice",
      "name": "OpenRouteService",
      "url": "https://openrouteservice.org/pricing",
      "cost": 10,
      "breakdown": {
        "geocoding": {
          "requests": 50000,
          "unit_price": 1,
          "free_tier": 40000,
          "billed_requests": 10000,
          "cost": 10
        },
        "routing": {
          "requests": 10000,
          "unit_price": 1.5,
          "free_tier": 40000,
          "billed_requests": 0,
          "cost": 0
        },
        ...
      }
    },
    ...
  ],
  "best_value": "openrouteservice",
  "total_cost": 10,
  "currency": "USD"
}
```

## Примеры использования

### Базовый пример с curl
```bash
curl -X POST http://localhost:8080/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 50000,
      "routing": 10000,
      "map_tiles_raster": 100000,
      "static_maps": 5000
    }
  }'
```

### Расчет без бесплатного кредита (не новый клиент)
```bash
curl -X POST http://localhost:8080/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 50000,
      "routing": 10000,
      "map_tiles_raster": 100000
    },
    "disable_new_customer_credit": true
  }'
```

### Получить список провайдеров
```bash
curl http://localhost:8080/providers | jq .
```

## Поддерживаемые провайдеры

1. **Google Maps Platform** - основной провайдер с $300 бесплатного кредита на первый месяц
2. **Mapbox** - альтернатива с собственной ценовой моделью и large free tiers

Примечание: опция `disable_new_customer_credit` актуальна для Google Maps (не применяется $300 кредит)

## Поддерживаемые API типы

- `geocoding` - геокодирование адресов
- `routing` - расчет маршрутов
- `map_tiles_raster` - растровые плитки карт
- `map_tiles_vector_2d` - 2D векторные плитки карт
- `map_tiles_vector_3d` - 3D векторные плитки карт
- `static_maps` - статические изображения карт
- `distance_matrix` - матрица расстояний
- `elevation` - высотные данные

## Структура проекта

```
calculator_api/
├── cmd/
│   └── calculator/
│       └── main.go              # Точка входа, HTTP сервер
├── internal/
│   ├── domain/
│   │   └── models.go            # Доменные модели
│   ├── handler/
│   │   └── calculator.go        # HTTP handlers
│   ├── usecase/
│   │   └── calculator.go        # Бизнес-логика калькуляции
│   └── pricing/
│       └── loader.go            # Загрузчик ценовых данных
├── pricing/
│   └── pricing.json             # Ценовые данные провайдеров
└── go.mod                        # Go модуль
```

## Ценовая модель

Калькулятор учитывает:

- **Free Tier** - бесплатные квоты каждого провайдера
- **Pay-as-you-go** - плата за каждые 1000 запросов сверх лимита
- **Free Credits** - бесплатные месячные кредиты для новых клиентов (например, Google Maps $300)

### Опция `disable_new_customer_credit`

По умолчанию калькулятор применяет бесплатные кредиты новых клиентов. Если вы хотите узнать реальную стоимость без кредитов (например, для продления подписки после первого месяца), используйте флаг `disable_new_customer_credit: true`.

## Разработка

### Добавление нового провайдера

1. Добавьте провайдера в `pricing/pricing.json`:

```json
{
  "provider_id": {
    "name": "Provider Name",
    "url": "https://provider.com/pricing",
    "apis": {
      "geocoding": {
        "unit": "request",
        "price_per_1000": 5,
        "free_tier": 10000
      },
      ...
    }
  }
}
```

2. Перезагрузите сервер

## Лицензия

MIT
