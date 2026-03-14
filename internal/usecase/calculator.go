package usecase

import (
	"fmt"
	"log/slog"
	"math"
	"sort"

	"calculator_api/internal/domain"
)

const (
	apiTypeNotSupported = "is not provided by %s"
)

var apiTypeNames = map[string]string{
	"geocoding":           "Geocoding",
	"routing":             "Routing",
	"map_tiles_raster":    "Map Tiles Raster",
	"map_tiles_vector_2d": "Map Tiles Vector 2D",
	"map_tiles_vector_3d": "Map Tiles Vector 3D",
	"static_maps":         "Static Maps",
	"static_street_view":  "Street View",
	"distance_matrix":     "Distance Matrix",
	"elevation":           "Elevation",
	"aerial_view":         "Aerial View",
}

type Calculator interface {
	Calculate(req *domain.CalculationRequest, pricing *domain.PricingData) *domain.CalculationResponse
}

type CalculatorImpl struct{}

func NewCalculator() Calculator {
	return &CalculatorImpl{}
}

func (c *CalculatorImpl) Calculate(req *domain.CalculationRequest, pricing *domain.PricingData) *domain.CalculationResponse {
	slog.Debug("Starting calculation", "providers_count", len(pricing.Providers), "api_requests_count", len(req.APIRequests), "disable_free_tier", req.DisableFreeTier)
	results := make([]domain.CalculationResult, 0, len(pricing.Providers))

	providersWithoutApi := []domain.CalculationResult{}

	for providerID, providerPricing := range pricing.Providers {
		slog.Debug("Calculating for provider", "provider_id", providerID, "provider_name", providerPricing.Name)
		result := c.calculateProvider(providerID, providerPricing, req.APIRequests, req.DisableNewCustomerCredit, req.DisableFreeTier)
		if len(result.Breakdown) != len(req.APIRequests) {
			providersWithoutApi = append(providersWithoutApi, result)
			continue
		}
		results = append(results, result)
		slog.Debug("Provider calculation completed", "provider_id", providerID, "cost", result.Cost)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Cost < results[j].Cost
	})

	bestValue := ""
	if len(results) > 0 {
		bestValue = results[0].Provider
		slog.Info("Best value determined", "provider", bestValue, "cost", results[0].Cost)
	}

	return &domain.CalculationResponse{
		Results:     append(results, providersWithoutApi...),
		BestValue:   bestValue,
		TotalCost:   calculateTotalCost(results),
		CurrencyUSD: pricing.Metadata.Currency,
	}
}

func (c *CalculatorImpl) calculateProvider(
	providerID string, providerPricing domain.ProviderPricing,
	apiRequests map[string]int, disableNewCustomerCredit bool, disableFreeTier bool,
) domain.CalculationResult {
	breakdown := make(map[string]domain.APICostBreakdown)
	totalCost := 0.0
	freeCredit := 0.0
	totalRequestsCount := 0
	notes := ""

	// Apply one-time free credit (e.g., Google trial credit)
	if !disableNewCustomerCredit && providerPricing.FreeCredit != nil && providerPricing.FreeCredit.Type == "one_time" {
		freeCredit = providerPricing.FreeCredit.AmountUSD
		slog.Debug("One-time free credit applied", "provider_id", providerID, "credit_amount", freeCredit)
		notes += "Includes one-time new customer credit"
	}

	// Check if this provider uses shared pool
	hasSharedPool := !disableFreeTier && providerPricing.MonthlyFreeTier != nil && providerPricing.MonthlyFreeTier.Type == "shared_pool"
	sharedPoolSize := 0
	if hasSharedPool {
		sharedPoolSize = providerPricing.MonthlyFreeTier.AmountRequests
		slog.Debug("Shared pool detected", "provider_id", providerID, "pool_size", sharedPoolSize)
	}

	// First pass: calculate all costs WITHOUT free tiers (for shared pool calculation)
	var rawCosts map[string]float64
	if hasSharedPool {
		rawCosts = make(map[string]float64)
	}

	// Calculate cost for each API
	for apiType, requestCount := range apiRequests {
		if apiPricing, exists := providerPricing.APIs[apiType]; exists {
			// Skip APIs that have no pricing (not supported by this provider)
			// An API is supported if it has either Tiers or a positive PricePer1000
			hasPrice := (apiPricing.PricePer1000 > 0) || (len(apiPricing.Tiers) > 0)

			if !hasPrice {
				slog.Debug("API not supported by provider",
					"provider_id", providerID,
					"api_type", apiType,
				)

				notes += fmt.Sprintf("%s not supported;\n", formatAPITypeName(apiType))
				continue
			}

			totalRequestsCount += requestCount

			// For shared pool: first calculate raw cost (no free tier)
			// For others: calculate with free tier
			useIndividualFreeTier := !hasSharedPool && !disableFreeTier

			if hasSharedPool {
				// Calculate raw cost for shared pool calculation
				rawCost := float64(requestCount) * apiPricing.PricePer1000 / 1000.0
				rawCosts[apiType] = rawCost
				costBreakdown := calculateAPICost(requestCount, apiPricing, false)
				breakdown[apiType] = costBreakdown
			} else {
				costBreakdown := calculateAPICost(requestCount, apiPricing, useIndividualFreeTier)
				breakdown[apiType] = costBreakdown
				totalCost += costBreakdown.Cost
			}

			slog.Debug("API cost calculated",
				"provider_id", providerID,
				"api_type", apiType,
				"requests", requestCount,
				"has_shared_pool", hasSharedPool,
			)
		} else {
			notes += fmt.Sprintf("%s not supported; ", formatAPITypeName(apiType))
		}
	}

	// For shared pool: apply pool and calculate final cost
	if hasSharedPool && sharedPoolSize > 0 {
		// Sum all raw costs
		totalRawCost := 0.0
		for _, cost := range rawCosts {
			totalRawCost += cost
		}

		// Apply shared pool: reduce the number of billed requests
		requestsAfterPool := totalRequestsCount - sharedPoolSize
		if requestsAfterPool < 0 {
			requestsAfterPool = 0
		}

		// Recalculate cost ONLY for requests beyond the pool
		if requestsAfterPool > 0 {
			// Ratio of requests to bill
			billRatio := float64(requestsAfterPool) / float64(totalRequestsCount)
			totalCost = totalRawCost * billRatio
		} else {
			// All requests covered by pool
			totalCost = 0
		}

		slog.Debug("Shared pool applied",
			"provider_id", providerID,
			"total_requests", totalRequestsCount,
			"pool_size", sharedPoolSize,
			"requests_after_pool", requestsAfterPool,
			"raw_cost", totalRawCost,
			"cost_after_pool", totalCost,
		)

		if notes != "" {
			notes += "; "
		}
		notes += fmt.Sprintf("Includes %d monthly free requests (shared pool)", sharedPoolSize)
	}

	// Apply one-time free credit
	totalCost = totalCost - freeCredit
	if totalCost < 0 {
		totalCost = 0
	}

	totalCost = math.Ceil(totalCost)

	if disableFreeTier {
		notes = "Free tier disabled"
	}

	slog.Debug("Provider calculation complete",
		"provider_id", providerID,
		"apis_count", len(breakdown),
		"total_requests", totalRequestsCount,
		"final_cost", totalCost,
	)

	return domain.CalculationResult{
		Provider:  providerID,
		Name:      providerPricing.Name,
		URL:       providerPricing.URL,
		Cost:      roundCost(totalCost),
		Breakdown: breakdown,
		Notes:     notes,
	}
}

func calculateAPICost(requests int, pricing domain.APIPricing, applyFreeTier bool) domain.APICostBreakdown {
	// Apply free tier first
	billedRequests := requests
	if applyFreeTier {
		billedRequests = requests - pricing.FreeTier
		if billedRequests < 0 {
			billedRequests = 0
		}
	}

	freeTier := pricing.FreeTier
	if !applyFreeTier {
		freeTier = 0
	}

	// Calculate cost using tiers (volume-based pricing) or fallback to single price
	var cost float64
	var unitPrice float64

	if len(pricing.Tiers) > 0 {
		// NEW: Use volume-based pricing tiers
		cost = calculateCostWithTiers(billedRequests, pricing.Tiers)
		// For display, show the first tier price (approximate)
		if len(pricing.Tiers) > 0 {
			unitPrice = pricing.Tiers[0].PricePer1000
		}
	} else {
		// LEGACY: Use single price per 1000
		cost = float64(billedRequests) * pricing.PricePer1000 / 1000.0
		unitPrice = pricing.PricePer1000
	}

	return domain.APICostBreakdown{
		Requests:       requests,
		UnitPrice:      unitPrice,
		FreeTier:       freeTier,
		BilledRequests: billedRequests,
		Cost:           roundCost(cost),
	}
}

// calculateCostWithTiers applies volume-based pricing from tiers
// Example: 600K requests with tiers:
//
//	Tier 1: 0-500k @ $5/1K = 500k * 0.005 = $2500
//	Tier 2: 500k+ @ $4/1K = 100k * 0.004 = $400
//	Total = $2900
func calculateCostWithTiers(billedRequests int, tiers []domain.PricingTier) float64 {
	if billedRequests == 0 {
		return 0
	}

	totalCost := 0.0
	remainingRequests := billedRequests

	for _, tier := range tiers {
		if remainingRequests <= 0 {
			break
		}

		// Calculate how many requests fall into this tier
		tierSize := tier.ToRequests - tier.FromRequests + 1
		if tier.ToRequests == 0 {
			// 0 means unlimited (last tier)
			tierSize = remainingRequests
		}

		requestsInThisTier := remainingRequests
		if requestsInThisTier > tierSize {
			requestsInThisTier = tierSize
		}

		// Calculate cost for this tier
		tierCost := float64(requestsInThisTier) * tier.PricePer1000 / 1000.0
		totalCost += tierCost

		remainingRequests -= requestsInThisTier
	}

	return totalCost
}

func calculateTotalCost(results []domain.CalculationResult) float64 {
	if len(results) == 0 {
		return 0
	}
	return results[0].Cost
}

func roundCost(cost float64) float64 {
	return float64(int(cost*100)) / 100.0
}

func formatAPITypeName(apiType string) string {
	if name, exists := apiTypeNames[apiType]; exists {
		return name
	}
	return apiType
}
