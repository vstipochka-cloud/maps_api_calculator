# 🛠️ IMPLEMENTATION ROADMAP - Ч то осталось и как это сделать

## 1️⃣ Distance Matrix Нормализация (КРИТИЧНО)

### Проблема
```
User input: "distance_matrix": 10000

Google биллирует:     10000 × $5/1K = $50
  Но на самом деле это может быть 100000 элементов!
  Правильная цена: 100000 × $5/1K = $500
  РАЗНИЦА: 10x!

Mapbox биллирует:     10000 × $2/1K = $20
  Это 1 запрос = 1 матрица
  Может быть 100K элементов в одной матрице

HERE биллирует:       10000 × $5/1K = $50
  Это "transactions" 
  Как они считают? Неясно.
```

### Решение
**Добавить в pricing.json:**
```json
{
  "api_normalization": {
    "distance_matrix": {
      "base_unit": "matrix_element",
      "definition": "Single element in distance matrix (origin-destination pair)",
      "notes": "Different providers bill differently for matrix APIs",
      "normalization_factors": {
        "google_maps": {
          "requests_to_elements": 100,
          "note": "1 request = ~100 elements (10×10 matrix typical)"
        },
        "mapbox": {
          "requests_to_elements": 1,
          "note": "1 request = elements you specify"
        },
        "here": {
          "requests_to_elements": 1,
          "note": "1 transaction = needs verification"
        }
      }
    }
  }
}
```

**Обновить interface:**
```go
type CalculationRequest struct {
  APIRequests map[string]int `json:"api_requests"`
  // NEW:
  MatrixSize  map[string]int `json:"matrix_size"`  // {"distance_matrix": 100} = 10×10
}
```

**Обновить логику:**
```go
// Для distance_matrix применить коэффициент нормализации
normalizedCount := requestCount
if apiType == "distance_matrix" && normalizationFactor > 1 {
  normalizedCount = requestCount * normalizationFactor
}

cost = float64(normalizedCount) * price / 1000
```

**Время:** 4-5 часов  
**Сложность:** Высокая

---

## 2️⃣ Volume Discounts (КРИТИЧНО для Enterprise)

### Проблема
```
Google на 10M запросов:
  Цена/1K:       $5
  Со скидкой:    $4 (20%)
  Экономия:      $10,000/месяц!
  Статус в системе: ИГНОРИРУЕТСЯ ❌

HERE на 5M запросов:
  Цена/1K:       $0.75
  Со скидкой:    $0.50 (33%)
  Экономия:      $1,250/месяц!
```

### Решение
**Добавить в pricing.json:**
```json
{
  "providers": {
    "google_maps": {
      "volume_discounts": [
        {
          "from_requests_per_month": 1000000,
          "to_requests_per_month": 10000000,
          "discount_percent": 10,
          "note": "Requires agreement"
        },
        {
          "from_requests_per_month": 10000000,
          "to_requests_per_month": 100000000,
          "discount_percent": 20,
          "note": "Enterprise agreement required"
        }
      ]
    }
  }
}
```

**Обновить модель:**
```go
type ProviderPricing struct {
  // ... existing
  VolumeDiscounts []VolumeDiscount `json:"volume_discounts"`
}

type VolumeDiscount struct {
  FromRequests    int     `json:"from_requests_per_month"`
  ToRequests      int     `json:"to_requests_per_month"`
  DiscountPercent float64 `json:"discount_percent"`
  Note            string  `json:"note"`
}
```

**Обновить логику:**
```go
// После базового расчета цены
appliedDiscount := 0.0
for _, discount := range providerPricing.VolumeDiscounts {
  if totalRequests >= discount.FromRequests && totalRequests < discount.ToRequests {
    appliedDiscount = float64(totalRequests) * float64(discount.DiscountPercent) / 100.0
  }
}
totalCost -= appliedDiscount
```

**Время:** 2-3 часа  
**Сложность:** Средняя

---

## 3️⃣ OpenRouteService (РАСШИРЕНИЕ)

### Данные
```json
{
  "openrouteservice": {
    "name": "OpenRouteService",
    "url": "https://openrouteservice.org/pricing",
    "free_credit": null,
    "monthly_free_tier": {
      "type": "freemium",
      "amount_requests": 10000000,
      "note": "10M/month free for development (unlimited for non-commercial)"
    },
    "apis": {
      "geocoding": {
        "unit": "request",
        "price_per_1000": 0.05,
        "free_tier": 10000000
      },
      "routing": {
        "unit": "request",
        "price_per_1000": 0.25,
        "free_tier": 10000000
      },
      "distance_matrix": {
        "unit": "request",
        "price_per_1000": 0.1,
        "free_tier": 10000000
      },
      "map_tiles_raster": {
        "unit": "tile",
        "price_per_1000": 0.1,
        "free_tier": 10000000
      },
      "map_tiles_vector_2d": {
        "unit": "tile",
        "price_per_1000": 0.1,
        "free_tier": 10000000
      },
      "elevation": {
        "unit": "request",
        "price_per_1000": 0.05,
        "free_tier": 10000000
      }
    }
  }
}
```

**Время:** 1 час  
**Сложность:** Низкая

---

## 4️⃣ Яндекс.Карты (ЛОКАЛЬНЫЙ РЫНОК)

### Данные
```json
{
  "yandex_maps": {
    "name": "Яндекс.Карты API",
    "url": "https://yandex.ru/dev/maps/",
    "currency": "RUB",
    "exchange_rate_to_usd": 0.011,
    "free_credit": null,
    "monthly_free_tier": {
      "type": "shared_pool",
      "amount_requests": 750000,
      "note": "750K requests/month free (across all APIs)"
    },
    "apis": {
      "geocoding": {
        "unit": "request",
        "price_per_1000": 30,  // RUB
        "free_tier": 750000
      },
      "routing": {
        "unit": "request",
        "price_per_1000": 50,  // RUB
        "free_tier": 750000
      },
      "map_tiles_raster": {
        "unit": "tile",
        "price_per_1000": 0,  // Included with subscription
        "free_tier": 750000
      },
      "distance_matrix": {
        "unit": "request",
        "price_per_1000": 0,  // Not available
        "free_tier": 0
      }
    }
  }
}
```

**Примечание:** Яндекс использует RUB, нужна конвертация в USD

**Время:** 1 час  
**Сложность:** Низкая

---

## 5️⃣ SLA/Provider Profile (Опционально для v2.0)

### Структура
```json
{
  "provider_profile": {
    "sla_uptime_percent": 99.95,
    "regions": ["US", "EU", "APAC", "Asia"],
    "rate_limit": "10,000 requests/sec",
    "support_level": "enterprise",
    "documentation": "Excellent",
    "api_version": "v1.4",
    "notes": "Market leader, most stable"
  }
}
```

**Время:** 1 час  
**Сложность:** Низкая

---

## Временная смета ВСЕГО

| Компонент | Часов | Приоритет | Сложность |
|-----------|-------|-----------|-----------|
| Distance Matrix | 4-5 | 🔴 Это важно | Высокая |
| Volume Discounts | 2-3 | 🔴 Это важно | Средняя |
| OpenRouteService | 1 | 🟡 Нужен | Низкая |
| Яндекс.Карты | 1 | 🟡 Нужен | Низкая |
| SLA Profile | 1 | 🟢 Nice | Низкая |
| Экспорт PDF/CSV | 2-3 | 🟢 Nice | Средняя |
| **ИТОГО** | **11-17** | - | - |

---

## Рекомендуемая фаза-за-фазой реализация

### **PHASE 1 (1 неделя) - КРИТИЧЕСКИЕ ИСПРАВЛЕНИЯ**
```
День 1-2:  Distance Matrix нормализация (4-5h)
День 3:    Volume Discounts (2-3h)
День 4:    Тестирование (1-2h)
День 5:    Документация (1h)
```

### **PHASE 2 (1 неделя) - РАСШИРЕНИЕ**
```
День 1:    OpenRouteService (1h)
День 2:    Яндекс.Карты (1h)
День 3:    Интеграция в систему (1h)
День 4-5:  Тестирование и документация (2h)
```

### **PHASE 3 (1 неделя) - ПОЛИРОВКА**
```
День 1:    SLA информация (1h)
День 2-3:  Экспорт функции (2-3h)
День 4-5:  Финальное тестирование (2h)
```

**ИТОГО:**  3 недели на полную реализацию всех функций

---

## Тестовые сценарии для проверки

### Test 1: Distance Matrix с нормализацией
```bash
curl -X POST http://localhost:8080/calculate \
  -d '{
    "api_requests": {"distance_matrix": 1000},
    "matrix_size": {"distance_matrix": 100}
  }'
# Expected: Google должен быть дороже (1000 × 100 = 100K элементов)
```

### Test 2: Enterprise с volume discounts
```bash
curl -X POST http://localhost:8080/calculate \
  -d '{"api_requests": {"geocoding": 50000000}}'
# Expected: Google должен быть дешевле с 20% скидкой
```

### Test 3: OpenRouteService сравнение
```bash
curl -X POST http://localhost:8080/calculate \
  -d '{"api_requests": {"geocoding": 1000000}}'
# Expected: ORS должен быть в топ 3 (дешевле Google)
```

### Test 4: Яндекс для RUB рынка
```bash
curl -X POST http://localhost:8080/calculate \
  -d '{"api_requests": {"geocoding": 100000}}'
# Expected: Яндекс должен быть как вариант, цены в USD
```

---

## Checklist для реализации

- [ ] **Phase 1**
  - [ ] Обновить модель для distance_matrix нормализации
  - [ ] Реализовать логику нормализации в calc
  - [ ] Добавить volume discounts структуру
  - [ ] Реализовать логику volume discounts
  - [ ] Обновить README с примерами
  - [ ] Протестировать на всех провайдерах

- [ ] **Phase 2**
  - [ ] Добавить OpenRouteService в pricing.json
  - [ ] Добавить Яндекс.Карты в pricing.json
  - [ ] Обновить примеры
  - [ ] Протестировать новых провайдеров

- [ ] **Phase 3**
  - [ ] Добавить SLA информацию (опционально)
  - [ ] Реализовать экспорт в PDF
  - [ ] Реализовать экспорт в CSV
  - [ ] Финальное тестирование

---

**Status: 🟡 Ready for Phase 1 implementation**

