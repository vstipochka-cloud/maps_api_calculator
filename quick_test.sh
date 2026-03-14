#!/bin/bash

# Quick Test Examples - Ready to run!
# Just copy and paste any of these commands

BASE_URL="http://localhost:8080"

# ============================================================================
# QUICK TESTS - Copy & Paste Ready
# ============================================================================

# 1. Health Check
# curl -s "$BASE_URL/health" | jq .

# 2. List Providers
# curl -s "$BASE_URL/providers" | jq '.providers[]'

# ============================================================================
# STARTUP EXAMPLES (Free tier focused)
# ============================================================================

# Tiny startup - 10K geocoding requests
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{"api_requests": {"geocoding": 10000}}' | \
  python3 << 'EOF'
import json
import sys
data = json.load(sys.stdin)
print("🚀 Tiny Startup - 10K Geocoding Requests\n")
for r in data['results']:
    print(f"  {r['name']}: ${r['cost']:.2f}")
print(f"\n  Best value: {data['best_value']}")
EOF

echo ""
echo "─" * 60
echo ""

# Small startup - Balanced Usage
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 100000,
      "routing": 50000,
      "map_tiles_vector_2d": 100000
    }
  }' | python3 << 'EOF'
import json
import sys
data = json.load(sys.stdin)
print("📈 Small Company - 250K Mixed Requests\n")
for r in data['results']:
    print(f"  {r['name']}: ${r['cost']:.2f}")
print(f"\n  Savings with Mapbox: ${data['results'][1]['cost'] - data['results'][0]['cost']:.2f}")
EOF

echo ""
echo "─" * 60
echo ""

# Scaling company - 1M requests
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 500000,
      "routing": 300000,
      "map_tiles_raster": 200000
    }
  }' | python3 << 'EOF'
import json
import sys
data = json.load(sys.stdin)
print("💰 Scaling Company - 1M Requests\n")
for r in data['results']:
    print(f"  {r['name']}: ${r['cost']:.2f}")
savings = data['results'][1]['cost'] - data['results'][0]['cost']
if savings > 0:
    pct = (savings / data['results'][1]['cost']) * 100
    print(f"\n  💡 Mapbox saves: ${savings:.2f}/month ({pct:.1f}%)")
EOF

echo ""
echo "─" * 60
echo ""

# Enterprise - Tile heavy
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "map_tiles_raster": 2000000,
      "map_tiles_vector_2d": 1000000,
      "geocoding": 300000
    }
  }' | python3 << 'EOF'
import json
import sys
data = json.load(sys.stdin)
print("🏢 Enterprise - 3.3M Tile Requests\n")
for r in data['results']:
    print(f"  {r['name']}: ${r['cost']:.2f}")
savings = data['results'][1]['cost'] - data['results'][0]['cost']
if savings > 0:
    pct = (savings / data['results'][1]['cost']) * 100
    print(f"\n  💡 Mapbox saves: ${savings:.2f}/month ({pct:.1f}%)")
EOF

echo ""
echo "─" * 60
echo ""

# ============================================================================
# SPECIAL CASES
# ============================================================================

# New Customer vs Existing - $300 credit impact
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 200000,
      "routing": 150000
    },
    "disable_new_customer_credit": false
  }' | python3 << 'EOF'
import json
import sys
data = json.load(sys.stdin)
gm = [r for r in data['results'] if r['provider'] == 'google_maps'][0]
print("💳 NEW CUSTOMER - Google Maps with $300 credit\n")
print(f"  Google Maps: ${gm['cost']:.2f} ({gm['notes']})")
EOF

echo ""

curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "geocoding": 200000,
      "routing": 150000
    },
    "disable_new_customer_credit": true
  }' | python3 << 'EOF'
import json
import sys
data = json.load(sys.stdin)
gm = [r for r in data['results'] if r['provider'] == 'google_maps'][0]
print("🔄 EXISTING CUSTOMER - Google Maps WITHOUT credit\n")
print(f"  Google Maps: ${gm['cost']:.2f}")
print(f"\n  💰 New customer credit value: $300/month")
EOF

echo ""
echo "─" * 60
echo ""

# Ride-sharing use case
curl -s -X POST "$BASE_URL/calculate" \
  -H "Content-Type: application/json" \
  -d '{
    "api_requests": {
      "routing": 2000000,
      "distance_matrix": 500000,
      "geocoding": 300000
    }
  }' | python3 << 'EOF'
import json
import sys
data = json.load(sys.stdin)
print("🚗 RIDESHARE APP - High Routing Volume (2.8M)\n")
for r in data['results']:
    print(f"  {r['name']}: ${r['cost']:.2f}")
EOF
