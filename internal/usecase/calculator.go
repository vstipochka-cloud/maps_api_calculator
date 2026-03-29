package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	"calculator_api/internal/domain"
)

const (
	apiTypeNotSupported  = "is not provided by %s"
	usdToRubExchangeRate = 80.0 // Approximate exchange rate
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

type CalculatorImpl struct {
	converter CurrencyConverter
}

func NewCalculator(c CurrencyConverter) Calculator {
	return &CalculatorImpl{
		converter: c,
	}
}

func (c *CalculatorImpl) Calculate(req *domain.CalculationRequest, pricing *domain.PricingData) *domain.CalculationResponse {
	slog.Debug("Starting calculation", "providers_count", len(pricing.Providers), "api_requests_count", len(req.APIRequests), "disable_free_tier", req.DisableFreeTier, "currency", req.Currency)
	results := make([]domain.CalculationResult, 0, len(pricing.Providers))

	providersWithoutApi := []domain.CalculationResult{}

	for providerID, providerPricing := range pricing.Providers {
		slog.Debug("Calculating for provider", "provider_id", providerID, "provider_name", providerPricing.Name)
		result := c.calculateProvider(providerID, providerPricing, req.APIRequests, req.MatrixParams, req.DisableNewCustomerCredit, req.DisableFreeTier)
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

	totalCost := calculateTotalCost(results)

	// Apply currency conversion if needed
	exchangeRate := 1.0
	if req.Currency == "RUB" {
		rate, err := c.converter.Convert(context.Background(), req.Currency)
		if err != nil {
			slog.Error("failed to convert", slog.Any("error", err))
			exchangeRate = usdToRubExchangeRate
		} else {
			exchangeRate = rate
		}
		// Convert all results
		for i := range results {
			results[i].ConvertedCost = results[i].Cost * exchangeRate
			// Also convert breakdown
			for apiType, breakdown := range results[i].Breakdown {
				breakdown.ConvertedCost = breakdown.Cost * exchangeRate
				results[i].Breakdown[apiType] = breakdown
			}
		}
		// Convert provided without API results
		for i := range providersWithoutApi {
			providersWithoutApi[i].ConvertedCost = providersWithoutApi[i].Cost * exchangeRate
			for apiType, breakdown := range providersWithoutApi[i].Breakdown {
				breakdown.ConvertedCost = breakdown.Cost * exchangeRate
				providersWithoutApi[i].Breakdown[apiType] = breakdown
			}
		}
	}

	response := &domain.CalculationResponse{
		Results:       append(results, providersWithoutApi...),
		BestValue:     bestValue,
		TotalCost:     totalCost,
		BaseCurrency:  "USD",
		Currency:      req.Currency,
		ExchangeRate:  exchangeRate,
		ConvertedCost: totalCost * exchangeRate,
	}

	slog.Debug("Calculation complete", "currency", response.Currency, "exchange_rate", response.ExchangeRate)

	return response
}

func (c *CalculatorImpl) calculateProvider(
	providerID string, providerPricing domain.ProviderPricing,
	apiRequests map[string]int, matrixParams *domain.MatrixParams, disableNewCustomerCredit bool, disableFreeTier bool,
) domain.CalculationResult {
	// Check if ANY API has its own license tiers (for providers like Яндекс with per-API pricing)
	hasPerAPIPricing := false
	for _, apiPricing := range providerPricing.APIs {
		if len(apiPricing.LicenseTiers) > 0 {
			hasPerAPIPricing = true
			break
		}
	}

	// Handle annual license pricing with per-API tiers (e.g., Яндекс Карты)
	if hasPerAPIPricing {
		return c.calculatePerAPILicenseProvider(providerID, providerPricing, apiRequests)
	}

	// Handle provider-level annual license pricing (e.g., older Яндекс structure)
	if (providerPricing.PricingModel == "annual_license" || providerPricing.PricingModel == "annual_license_with_overage" || providerPricing.PricingModel == "annual_dau_license_with_overage") && len(providerPricing.LicenseTiers) > 0 {
		return c.calculateAnnualLicenseProvider(providerID, providerPricing, apiRequests)
	}

	// Standard pay-as-you-go model (default)
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

			// Calculate actual billable requests (might be matrix elements)
			billableRequests := requestCount
			if apiPricing.CalculateMatrixElements && apiType == "distance_matrix" {
				billableRequests = calculateMatrixElements(requestCount, matrixParams)
				slog.Debug("Using matrix element calculation",
					"provider_id", providerID,
					"api_type", apiType,
					"input_requests", requestCount,
					"billable_requests", billableRequests,
				)
			}

			if hasSharedPool {
				// Calculate raw cost for shared pool calculation
				rawCost := float64(billableRequests) * apiPricing.PricePer1000 / 1000.0
				rawCosts[apiType] = rawCost
				costBreakdown := calculateAPICost(billableRequests, apiPricing, false)
				breakdown[apiType] = costBreakdown
			} else {
				costBreakdown := calculateAPICost(billableRequests, apiPricing, useIndividualFreeTier)
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

// calculateMatrixElements converts API requests to matrix elements for providers that charge by elements
// For Google Maps Distance Matrix: elements = requests × origins × destinations
// Example: 10 requests with 3 origins and 5 destinations = 10 × 3 × 5 = 150 elements
func calculateMatrixElements(requests int, matrixParams *domain.MatrixParams) int {
	if matrixParams == nil || matrixParams.OriginsCount == 0 || matrixParams.DestinationsCount == 0 {
		// Return requests as-is if matrix params are missing
		slog.Debug("Matrix params not provided or incomplete, treating requests as elements", "requests", requests)
		return requests
	}

	elements := requests * matrixParams.OriginsCount * matrixParams.DestinationsCount
	slog.Debug("Matrix elements calculated",
		"requests", requests,
		"origins", matrixParams.OriginsCount,
		"destinations", matrixParams.DestinationsCount,
		"total_elements", elements,
	)
	return elements
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

// calculatePerAPILicenseProvider handles providers where each API has its own license tiers (e.g., Яндекс Карты)
func (c *CalculatorImpl) calculatePerAPILicenseProvider(
	providerID string, providerPricing domain.ProviderPricing, apiRequests map[string]int,
) domain.CalculationResult {
	breakdown := make(map[string]domain.APICostBreakdown)
	totalCost := 0.0
	notes := ""

	exchangeRate, err := c.converter.Convert(context.Background(), "RUB")
	if err != nil {
		slog.Warn("Failed to get exchange rate, using fallback", "error", err)
		exchangeRate = 95.0
	}

	for apiType, requestCount := range apiRequests {
		if apiPricing, exists := providerPricing.APIs[apiType]; exists {
			if !apiPricing.Supported {
				notes += fmt.Sprintf("%s not supported; ", formatAPITypeName(apiType))
				continue
			}

			if len(apiPricing.LicenseTiers) == 0 {
				notes += fmt.Sprintf("%s has no pricing; ", formatAPITypeName(apiType))
				continue
			}

			// Calculate daily requests for this API
			dailyRequests := requestCount / 30
			if requestCount%30 > 0 {
				dailyRequests++
			}

			// Find best tier for this API
			var bestTier *domain.LicenseTier
			bestCost := math.MaxFloat64

			for i := range apiPricing.LicenseTiers {
				tier := &apiPricing.LicenseTiers[i]

				// Cost of full license for this tier (monthly)
				cost1 := tier.AnnualPriceRub / exchangeRate / 12.0

				// Cost if we use lower tier + overage
				var cost2 float64 = math.MaxFloat64
				if tier.OveragePerRub > 0 && dailyRequests > tier.DailyLimit {
					overageDaily := dailyRequests - tier.DailyLimit
					overageMonthly := overageDaily * 30
					overageRubMonthly := (float64(overageMonthly) / 1000.0) * tier.OveragePerRub
					cost2 = cost1 + (overageRubMonthly / exchangeRate)
				}

				var tierCost float64
				if cost2 > 0 && cost2 < cost1 {
					tierCost = cost2
				} else {
					tierCost = cost1
				}

				// Select this tier if it fits the daily limit or if it's the only option with overage
				if (tier.DailyLimit >= dailyRequests || tier.OveragePerRub > 0) && tierCost < bestCost {
					bestTier = tier
					bestCost = tierCost
				}
			}

			if bestTier == nil && len(apiPricing.LicenseTiers) > 0 {
				bestTier = &apiPricing.LicenseTiers[len(apiPricing.LicenseTiers)-1]
				bestCost = bestTier.AnnualPriceRub / exchangeRate / 12.0
			}

			if bestTier != nil {
				totalCost += bestCost
				displayName := formatAPITypeName(apiType)
				if apiPricing.DisplayName != "" {
					displayName = apiPricing.DisplayName
				}

				slog.Debug("Per-API cost calculated",
					"provider_id", providerID,
					"api_type", apiType,
					"display_name", displayName,
					"daily_requests", dailyRequests,
					"tier_daily_limit", bestTier.DailyLimit,
					"monthly_cost_usd", bestCost,
				)

				breakdown[apiType] = domain.APICostBreakdown{
					Requests:    requestCount,
					DisplayName: displayName,
					Cost:        bestCost,
				}
			}
		} else {
			notes += fmt.Sprintf("%s not supported; ", formatAPITypeName(apiType))
		}
	}

	notes = strings.TrimRight(notes, "; ")

	return domain.CalculationResult{
		Provider:  providerID,
		Name:      providerPricing.Name,
		URL:       providerPricing.URL,
		Cost:      roundCost(totalCost),
		Breakdown: breakdown,
		Notes:     notes,
	}
}

func (c *CalculatorImpl) calculateAnnualLicenseProvider(
	providerID string, providerPricing domain.ProviderPricing, apiRequests map[string]int,
) domain.CalculationResult {
	breakdown := make(map[string]domain.APICostBreakdown)

	totalMonthlyRequests := 0
	for _, count := range apiRequests {
		totalMonthlyRequests += count
	}

	dailyRequestsNeeded := totalMonthlyRequests / 30
	if totalMonthlyRequests%30 > 0 {
		dailyRequestsNeeded++
	}

	slog.Debug("Annual license calculation",
		"provider_id", providerID,
		"monthly_requests", totalMonthlyRequests,
		"daily_requests_needed", dailyRequestsNeeded,
	)

	exchangeRate, err := c.converter.Convert(context.Background(), "RUB")
	if err != nil {
		slog.Warn("Failed to get exchange rate, using fallback", "error", err)
		exchangeRate = 95.0
	}

	var bestOption *domain.LicenseTier
	bestCost := math.MaxFloat64
	var bestNote string

	for i := range providerPricing.LicenseTiers {
		tier := &providerPricing.LicenseTiers[i]

		cost1 := tier.AnnualPriceRub / exchangeRate / 12.0
		var option1Note string

		var cost2 float64 = math.MaxFloat64
		var option2Note string

		if tier.OveragePerRub > 0 && dailyRequestsNeeded > tier.DailyLimit {
			overageDaily := dailyRequestsNeeded - tier.DailyLimit
			overageMonthly := overageDaily * 30
			overageRubMonthly := (float64(overageMonthly) / 1000.0) * tier.OveragePerRub
			cost2 = cost1 + (overageRubMonthly / exchangeRate)
			option2Note = fmt.Sprintf("Лицензия ($%.2f) + переплата за %d запросов/день ($%.2f) = $%.2f",
				cost1, overageDaily, overageRubMonthly/exchangeRate, cost2)
			option1Note = fmt.Sprintf("Полная лицензия до %d запросов/день = $%.2f", tier.DailyLimit, cost1)
		}

		var tierCost float64
		var tierNote string

		if cost2 > 0 && cost2 < cost1 {
			tierCost = cost2
			tierNote = option2Note
		} else {
			tierCost = cost1
			tierNote = option1Note
		}

		if (tier.DailyLimit >= dailyRequestsNeeded || tier.OveragePerRub > 0) && tierCost < bestCost {
			bestOption = tier
			bestCost = tierCost
			bestNote = tierNote
		}
	}

	if bestOption == nil && len(providerPricing.LicenseTiers) > 0 {
		bestOption = &providerPricing.LicenseTiers[len(providerPricing.LicenseTiers)-1]
		bestCost = bestOption.AnnualPriceRub / exchangeRate / 12.0
	}

	if bestOption == nil {
		return domain.CalculationResult{
			Provider:  providerID,
			Name:      providerPricing.Name,
			URL:       providerPricing.URL,
			Cost:      0,
			Breakdown: breakdown,
			Notes:     "No pricing tiers available",
		}
	}

	slog.Debug("License tier selected",
		"provider_id", providerID,
		"daily_limit", bestOption.DailyLimit,
		"monthly_cost_usd", bestCost,
	)

	notes := fmt.Sprintf("Annual license ₽%.0f/year: %s", bestOption.AnnualPriceRub, bestNote)

	for apiType, requestCount := range apiRequests {
		if apiPricing, exists := providerPricing.APIs[apiType]; exists {
			if apiPricing.Supported {
				breakdown[apiType] = domain.APICostBreakdown{
					Requests:       requestCount,
					UnitPrice:      0,
					FreeTier:       0,
					BilledRequests: 0,
					Cost:           0,
				}
			} else {
				notes = fmt.Sprintf("%s not supported; %s", formatAPITypeName(apiType), notes)
			}
		}
	}

	return domain.CalculationResult{
		Provider:  providerID,
		Name:      providerPricing.Name,
		URL:       providerPricing.URL,
		Cost:      roundCost(bestCost),
		Breakdown: breakdown,
		Notes:     notes,
	}
}

func calculateTotalCost(results []domain.CalculationResult) float64 {
	if len(results) == 0 {
		return 0
	}
	return results[0].Cost
}

func roundCost(cost float64) float64 {
	return math.Ceil(float64(int(cost*100)) / 100.0)
}

func formatAPITypeName(apiType string) string {
	if name, exists := apiTypeNames[apiType]; exists {
		return name
	}
	return apiType
}
