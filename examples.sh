#!/bin/bash

# примеры запросов к API калькулятора

BASE_URL="http://localhost:8080"

echo "=== 1. Проверка здоровья сервера ==="
curl -s "$BASE_URL/health" | jq .
echo ""

echo "=== 2. Получить список доступных провайдеров ==="
curl -s "$BASE_URL/providers" | jq .
echo ""

echo "=== 3. Расчет для малого проекта (10K запросов) ==="
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 10000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'
echo ""

echo "=== 4. Расчет для среднего проекта (100K запросов в месяц) ==="
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 50000,
      "routing": 20000,
      "map_tiles_raster": 30000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'
echo ""

echo "=== 5. Расчет для крупного проекта (1M+ запросов) ==="
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 500000,
      "routing": 200000,
      "map_tiles_raster": 300000,
      "distance_matrix": 100000,
      "static_maps": 50000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'
echo ""

echo "=== 6. Полный расчет для всех доступных API типов ==="
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 50000,
      "routing": 10000,
      "map_tiles_raster": 100000,
      "map_tiles_vector_2d": 50000,
      "static_maps": 5000,
      "distance_matrix": 20000,
      "elevation": 10000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost, best: .provider == .best_value}'
