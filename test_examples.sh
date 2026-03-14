#!/bin/bash

# Comprehensive test examples for API Calculator
# Run the calculator server first: ./calculator

BASE_URL="http://localhost:8080"
DIVIDER="============================================================================"

echo "$DIVIDER"
echo "API Calculator - Comprehensive Test Suite"
echo "$DIVIDER"
echo ""

# Function to format output
test_case() {
    echo ""
    echo ">>> $1"
    echo "$DIVIDER"
}

# ============================================================================
# SECTION 1: BASIC TESTS
# ============================================================================

test_case "1.1: Server Health Check"
curl -s "$BASE_URL/health" | jq .

test_case "1.2: Get Available Providers and API Types"
curl -s "$BASE_URL/providers" | jq '.providers[] | {id, name, url}'

test_case "1.3: List All Supported API Types"
curl -s "$BASE_URL/providers" | jq '.api_types'

# ============================================================================
# SECTION 2: STARTUP SCENARIOS
# ============================================================================

test_case "2.1: Tiny Startup - Only Geocoding (10K requests)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{"api_requests": {"geocoding": 10000}}' | \
  jq '.results[] | {provider: .name, cost: .cost, notes: .notes}'

test_case "2.2: Small Startup - Basic Mapping (geocoding + maps)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 25000,
      "map_tiles_raster": 50000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "2.3: Early-Stage Startup - Full Stack (all services)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 50000,
      "routing": 30000,
      "map_tiles_raster": 70000,
      "map_tiles_vector_2d": 40000,
      "static_maps": 20000,
      "distance_matrix": 15000,
      "elevation": 10000
    }
  }' | jq '.results[] | {provider: .name, cost, best: (.provider == "mapbox")}'

# ============================================================================
# SECTION 3: GROWTH SCENARIOS
# ============================================================================

test_case "3.1: Growing Company - Geocoding Heavy (500K)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 500000,
      "map_tiles_raster": 100000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost, monthly_savings_vs_google: (.name == "Mapbox" ? (5500 - .cost) | tostring + " USD" : null)}'

test_case "3.2: Medium Company - Balanced Usage (1M requests)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 300000,
      "routing": 200000,
      "map_tiles_raster": 300000,
      "static_maps": 200000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "3.3: Growing Company - Tile Heavy Usage"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "map_tiles_raster": 2000000,
      "map_tiles_vector_2d": 1000000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

# ============================================================================
# SECTION 4: ENTERPRISE SCENARIOS
# ============================================================================

test_case "4.1: Enterprise - Routing Heavy (high volume)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "routing": 3000000,
      "distance_matrix": 500000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "4.2: Enterprise - Geocoding + Routing (2M combined)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 1000000,
      "routing": 1000000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "4.3: Large Scale - All Services Heavy Usage (5M+ requests)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 2000000,
      "routing": 1500000,
      "map_tiles_raster": 1000000,
      "map_tiles_vector_2d": 500000,
      "static_maps": 300000,
      "distance_matrix": 200000,
      "elevation": 100000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "4.4: Ultra High Volume - Tile-Heavy Enterprise"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "map_tiles_raster": 5000000,
      "map_tiles_vector_2d": 3000000,
      "geocoding": 500000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

# ============================================================================
# SECTION 5: SPECIFIC USE CASES
# ============================================================================

test_case "5.1: Ride-Sharing App (routing + geocoding heavy)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "routing": 2000000,
      "distance_matrix": 500000,
      "geocoding": 300000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "5.2: Delivery Service (distance matrix + routing)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "distance_matrix": 1000000,
      "routing": 800000,
      "geocoding": 200000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "5.3: Real Estate Platform (geocoding + static maps)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 600000,
      "static_maps": 400000,
      "map_tiles_vector_2d": 200000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "5.4: Analytics Platform (map visualization)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "map_tiles_raster": 1500000,
      "map_tiles_vector_2d": 1000000,
      "static_maps": 500000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "5.5: Fleet Management (routing + geocoding + elevation)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "routing": 1200000,
      "geocoding": 400000,
      "elevation": 300000,
      "map_tiles_raster": 500000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

# ============================================================================
# SECTION 6: CREDIT COMPARISON
# ============================================================================

test_case "6.1: WITH New Customer Credit ($300 Google Maps)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 100000,
      "routing": 100000,
      "map_tiles_raster": 100000
    },
    "disable_new_customer_credit": false
  }' | jq '.results[] | {provider: .name, cost: .cost, credit: .notes}'

test_case "6.2: WITHOUT New Customer Credit (existing customer)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 100000,
      "routing": 100000,
      "map_tiles_raster": 100000
    },
    "disable_new_customer_credit": true
  }' | jq '.results[] | {provider: .name, cost: .cost, credit: .notes}'

test_case "6.3: Cost Delta - New vs Existing Customer (500K requests)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 250000,
      "routing": 150000,
      "map_tiles_raster": 100000
    },
    "disable_new_customer_credit": false
  }' | jq '.results[] | select(.provider == "google_maps") | {provider: .name, cost_with_credit: .cost}' > /tmp/with_credit.json

curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 250000,
      "routing": 150000,
      "map_tiles_raster": 100000
    },
    "disable_new_customer_credit": true
  }' | jq '.results[] | select(.provider == "google_maps") | {provider: .name, cost_without_credit: .cost}' > /tmp/without_credit.json

echo "Google Maps: New Customer vs Existing"
echo "With Credit:    $(jq '.cost_with_credit' /tmp/with_credit.json)"
echo "Without Credit: $(jq '.cost_without_credit' /tmp/without_credit.json)"
echo "Difference:     \$$(( $(jq '.cost_without_credit' /tmp/without_credit.json | cut -d. -f1) - $(jq '.cost_with_credit' /tmp/with_credit.json | cut -d. -f1) ))"

# ============================================================================
# SECTION 7: EDGE CASES & EDGE SCENARIOS
# ============================================================================

test_case "7.1: Minimum Request (1 request)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{"api_requests": {"geocoding": 1}}' | \
  jq '.results[] | {provider: .name, cost: .cost}'

test_case "7.2: Single API Type - Large Volume"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{"api_requests": {"geocoding": 10000000}}' | \
  jq '.results[] | {provider: .name, cost: .cost}'

test_case "7.3: Exact Free Tier Boundary (100K geocoding requests)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{"api_requests": {"geocoding": 100000}}' | \
  jq '.results[] | {provider: .name, cost: .cost}'

test_case "7.4: Just Beyond Free Tier (100,001 geocoding requests)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{"api_requests": {"geocoding": 100001}}' | \
  jq '.results[] | {provider: .name, cost: .cost}'

test_case "7.5: Multiple APIs at Free Tier Limits"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 100000,
      "routing": 100000,
      "map_tiles_raster": 750000,
      "map_tiles_vector_2d": 200000,
      "static_maps": 50000,
      "distance_matrix": 100000,
      "elevation": 100000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost, all_free: (.cost == 0)}'

# ============================================================================
# SECTION 8: DETAILED BREAKDOWN
# ============================================================================

test_case "8.1: Detailed Breakdown - Medium Company (with notes)"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 200000,
      "routing": 100000,
      "map_tiles_raster": 300000
    }
  }' | jq '.results[] | {
    provider: .name,
    total_cost: .cost,
    breakdown: .breakdown | to_entries[] | {
      api: .key,
      requests: .value.requests,
      free_tier: .value.free_tier,
      billed: .value.billed_requests,
      price_per_1000: .value.unit_price,
      cost: .value.cost
    }
  }'

test_case "8.2: Comparison Summary - Best Value Analysis"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 500000,
      "routing": 300000,
      "map_tiles_vector_2d": 400000
    }
  }' | jq '{
    best_value: .best_value,
    total_comparison: [.results[] | {provider: .name, cost: .cost}],
    savings: (.results[] | select(.provider == .results[0].provider if .cost < .results[1].cost then .provider else .results[1].provider end) | (.results[1].cost - .results[0].cost))
  }'

# ============================================================================
# SECTION 9: REAL-WORLD SCENARIOS WITH COMMENTARY
# ============================================================================

test_case "9.1: Portfolio: SaaS Company with Multiple Services"
echo "Company A: Mapping + Delivery = Routing Heavy"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "routing": 2500000,
      "geocoding": 600000,
      "distance_matrix": 800000,
      "map_tiles_raster": 400000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost, recommendation: "Excellent for routing-heavy apps"}'

test_case "9.2: Portfolio: Mobile App with Map Display"
echo "Company B: Display Maps + User Location = Tile + Geocoding Heavy"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "map_tiles_vector_2d": 3000000,
      "map_tiles_raster": 2000000,
      "geocoding": 400000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

test_case "9.3: Portfolio: Location Intelligence Platform"
echo "Company C: Analytics + Search = Matrix + Elevation Heavy"
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "distance_matrix": 2000000,
      "elevation": 500000,
      "geocoding": 300000,
      "static_maps": 100000
    }
  }' | jq '.results[] | {provider: .name, cost: .cost}'

# ============================================================================
# SUMMARY
# ============================================================================

echo ""
echo "$DIVIDER"
echo "Test Suite Complete!"
echo "$DIVIDER"
echo ""
echo "Key Findings:"
echo "- Mapbox: Aggressive free tiers, great for startups"
echo "- Google Maps: High baseline cost, but includes \$300 new customer credit"
echo "- Crossover point: Around 500K-1M requests monthly"
echo ""
echo "For more details, check the server logs:"
echo "  tail -f /tmp/calculator.log"
echo ""
