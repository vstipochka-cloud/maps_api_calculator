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

	// Apply currency conversion if needed (any currency other than USD)
	slog.Info("AAAAAA exchange")
	exchangeRate := 1.0
	if req.Currency != "" && req.Currency != "USD" {
		rate, err := c.converter.Convert(context.Background(), req.Currency)
		if err != nil {
			slog.Error("failed to convert", slog.Any("error", err))
			// fallback to 1.0 (no conversion) on error to avoid wildly incorrect multipliers
			exchangeRate = 1.0
		} else {
			exchangeRate = rate
		}

		// Convert all results: convert per-API breakdowns first and sum them
		for i := range results {
			sumConv := 0.0
			for apiType, breakdown := range results[i].Breakdown {
				breakdown.ConvertedCost = roundCurrency(breakdown.Cost * exchangeRate)
				results[i].Breakdown[apiType] = breakdown
				sumConv += breakdown.ConvertedCost
			}
			// Use summed converted breakdowns to set ConvertedCost for consistency
			results[i].ConvertedCost = roundCurrency(sumConv)
		}

		// Convert provided without API results similarly
		for i := range providersWithoutApi {
			sumConv := 0.0
			for apiType, breakdown := range providersWithoutApi[i].Breakdown {
				breakdown.ConvertedCost = roundCurrency(breakdown.Cost * exchangeRate)
				providersWithoutApi[i].Breakdown[apiType] = breakdown
				sumConv += breakdown.ConvertedCost
			}
			providersWithoutApi[i].ConvertedCost = roundCurrency(sumConv)
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
		// Skip APIs with zero requests — don't charge licenses for unused APIs
		if requestCount == 0 {
			continue
		}

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
				billableRequests = calculateMatrixElements(providerID, requestCount, matrixParams)
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
		Provider: providerID,
		Name:     providerPricing.Name,
		URL:      providerPricing.URL,
		Cost:     roundCost(totalCost),
		PerRequest: func() float64 {
			if totalRequestsCount > 0 {
				return roundCost(totalCost / float64(totalRequestsCount))
			}
			return 0
		}(),
		Breakdown: breakdown,
		Notes:     notes,
	}
}

// calculateMatrixElements converts API requests to billable transactions/elements for providers
// Provider-specific rules are applied here. Default behavior (Google Maps style):
// elements = requests × origins × destinations
// HERE Technologies uses a different rule documented below.
func calculateMatrixElements(providerID string, requests int, matrixParams *domain.MatrixParams) int {
	if matrixParams == nil || matrixParams.OriginsCount == 0 || matrixParams.DestinationsCount == 0 {
		// Return requests as-is if matrix params are missing
		slog.Debug("Matrix params not provided or incomplete, treating requests as elements", "requests", requests, "provider", providerID)
		return requests
	}

	// HERE Technologies: "1 Transaction" counting rule
	// - If either Starting Points or Destination Points < 5:
	//     transactions = starting_points * destination_points
	// - If both Starting Points and Destination Points >= 5:
	//     transactions = 5 * max(starting_points, destination_points)
	// The provider bills per 1,000 transactions according to tiers.
	if providerID == "here" {
		origins := matrixParams.OriginsCount
		destinations := matrixParams.DestinationsCount
		var perRequestTransactions int

		if origins < 5 || destinations < 5 {
			perRequestTransactions = origins * destinations
		} else {
			// both >= 5
			if origins > destinations {
				perRequestTransactions = 5 * origins
			} else {
				perRequestTransactions = 5 * destinations
			}
		}

		total := requests * perRequestTransactions
		slog.Debug("HERE matrix transactions calculated",
			"provider", providerID,
			"requests", requests,
			"origins", origins,
			"destinations", destinations,
			"per_request_transactions", perRequestTransactions,
			"total_transactions", total,
		)
		return total
	}

	// Default: Google Maps style element calculation
	elements := requests * matrixParams.OriginsCount * matrixParams.DestinationsCount
	slog.Debug("Matrix elements calculated",
		"provider", providerID,
		"requests", requests,
		"origins", matrixParams.OriginsCount,
		"destinations", matrixParams.DestinationsCount,
		"total_elements", elements,
	)
	return elements
}

func calculateAPICost(requests int, pricing domain.APIPricing, applyFreeTier bool) domain.APICostBreakdown {
	// Determine billed requests and displayed free tier.
	// If volume-based tiers are defined, those tiers typically include any free/zero-priced ranges
	// (e.g., 0-10k @ $0). In that case, pass the full request count into tier calculation and
	// let the tiers apply the free portion. If no tiers exist, apply the individual free tier
	// by subtracting it from the request count.
	billedRequests := requests
	freeTier := pricing.FreeTier
	if !applyFreeTier {
		// Free tier disabled: show zero free and bill full requests
		freeTier = 0
	} else {
		if len(pricing.Tiers) == 0 {
			// No tiers to handle free ranges, subtract free tier explicitly
			billedRequests = requests - pricing.FreeTier
			if billedRequests < 0 {
				billedRequests = 0
			}
		} else {
			// Tiers exist and may already include free ranges; use full requests for tier calc
			billedRequests = requests
		}
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
		var tierSize int
		if tier.ToRequests == 0 {
			// 0 means unlimited (last tier)
			tierSize = remainingRequests
		} else {
			// Treat ranges as [from_requests, to_requests] but interpret size as
			// to - from (not +1) so that a range 0-10000 represents 10000 units.
			tierSize = tier.ToRequests - tier.FromRequests
			if tierSize < 0 {
				tierSize = 0
			}
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
		exchangeRate = fallbackFor("RUB")
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
			// Use 30 days in month for monthly <-> daily conversions
			daysInMonth := 30

			dailyRequests := requestCount / daysInMonth
			if requestCount%daysInMonth > 0 {
				dailyRequests++
			}

			// Handle free daily limit metadata (e.g., Yandex free daily usage)
			displayName := formatAPITypeName(apiType)
			if apiPricing.DisplayName != "" {
				displayName = apiPricing.DisplayName
			}

			// If provider offers a free daily limit and the user's daily requests are within it, cost is zero
			if apiPricing.FreeDailyLimit > 0 && dailyRequests <= apiPricing.FreeDailyLimit {
				breakdown[apiType] = domain.APICostBreakdown{
					Requests:    requestCount,
					DisplayName: displayName,
					Cost:        0,
				}
				notes += fmt.Sprintf("%s: within free daily limit %d; ", displayName, apiPricing.FreeDailyLimit)
				continue
			}

			// For Yandex: do not auto-select a license; instead, produce per-tier comparison (upgrade vs overage)
			if strings.ToLower(providerID) == "yandex" {
				// prepare comparison: base tier = first tier
				if len(apiPricing.LicenseTiers) == 0 {
					breakdown[apiType] = domain.APICostBreakdown{Requests: requestCount, DisplayName: displayName, Cost: 0}
					continue
				}

				base := apiPricing.LicenseTiers[0]
				baseMonthlyRub := base.AnnualPriceRub / 12.0
				baseMonthlyLocal := baseMonthlyRub / exchangeRate

				overageIfStayOnBaseRub := 0.0
				if base.OveragePerRub > 0 && dailyRequests > base.DailyLimit {
					overageDaily := dailyRequests - base.DailyLimit
					overageMonthly := overageDaily * daysInMonth
					overageIfStayOnBaseRub = (float64(overageMonthly) / 1000.0) * base.OveragePerRub
				}
				overageIfStayOnBaseLocal := overageIfStayOnBaseRub / exchangeRate

				// Determine whether staying on base + overage is allowed by policy
				stayAllowed := !apiPricing.OnExceedRequiresFullLicense

				slog.Debug("Yandex licensing policy", "api", apiType, "on_exceed_requires_full_license", apiPricing.OnExceedRequiresFullLicense, "daily_requests", dailyRequests)
				fmt.Printf("DEBUG YANDEX %s OnExceed=%v dailyRequests=%d\n", apiType, apiPricing.OnExceedRequiresFullLicense, dailyRequests)

				options := make([]domain.LicenseOption, 0, len(apiPricing.LicenseTiers))
				for i := range apiPricing.LicenseTiers {
					t := apiPricing.LicenseTiers[i]
					monthlyLicenseRub := t.AnnualPriceRub / 12.0
					monthlyLicenseLocal := monthlyLicenseRub / exchangeRate

					var cheaper string
					if monthlyLicenseLocal < (baseMonthlyLocal + overageIfStayOnBaseLocal) {
						cheaper = "upgrade"
					} else if monthlyLicenseLocal > (baseMonthlyLocal + overageIfStayOnBaseLocal) {
						cheaper = "overage"
					} else {
						cheaper = "equal"
					}

					// Only include tiers that fully cover required daily requests
					if t.DailyLimit < dailyRequests {
						// skip this tier — it cannot be used as a full license to cover current usage
						continue
					}

					options = append(options, domain.LicenseOption{
						Name:                   t.Name,
						DailyLimit:             t.DailyLimit,
						MonthlyLicenseRub:      monthlyLicenseRub,
						MonthlyLicenseLocal:    monthlyLicenseLocal,
						OverageIfStayOnBaseRub: overageIfStayOnBaseRub,
						OverageIfStayOnBase:    overageIfStayOnBaseLocal,
						Cheaper:                cheaper,
					})
				}

				// Start minimal candidate: if staying on base+overage is allowed, use that, otherwise set to +Inf
				stayMonthlyLocal := baseMonthlyLocal + overageIfStayOnBaseLocal
				fmt.Printf("DEBUG YANDEX %s baseMonthlyLocal=%.2f overageLocal=%.2f stayMonthlyLocal=%.2f stayAllowed=%v\n", apiType, baseMonthlyLocal, overageIfStayOnBaseLocal, stayMonthlyLocal, stayAllowed)
				for _, t := range apiPricing.LicenseTiers {
					if t.DailyLimit >= dailyRequests {
						monthlyLicenseLocal := (t.AnnualPriceRub / 12.0) / exchangeRate
						fmt.Printf("DEBUG YANDEX %s tier %s monthlyLicenseLocal=%.2f dailyLimit=%d\n", apiType, t.Name, monthlyLicenseLocal, t.DailyLimit)
					}
				}

				minLocal := math.Inf(1)
				if stayAllowed {
					minLocal = stayMonthlyLocal
				}

				// Consider only valid upgrade options (those included in options slice)
				for _, opt := range options {
					if opt.MonthlyLicenseLocal < minLocal {
						minLocal = opt.MonthlyLicenseLocal
					}
				}

				// If no valid option found (shouldn't happen), fallback to base monthly (without overage)
				if math.IsInf(minLocal, 1) {
					minLocal = baseMonthlyLocal
				}

				// Present options and set chosen cost to minimal for UI
				breakdown[apiType] = domain.APICostBreakdown{
					Requests:       requestCount,
					DisplayName:    displayName,
					Cost:           roundCost(minLocal),
					LicenseOptions: options,
				}
				// add minimal option to provider total (minLocal is in base currency units — USD)
				totalCost += minLocal
				continue
				// add minimal option to provider total (in local currency units are converted to USD here)
				totalCost += minLocal
				continue
			}
			// Non-Yandex providers: previous behavior (find best tier and add cost)
			var bestTier *domain.LicenseTier
			bestCost := math.MaxFloat64

			for i := range apiPricing.LicenseTiers {
				tier := &apiPricing.LicenseTiers[i]

				// Cost of full license for this tier (monthly) — only valid if the tier covers required daily requests
				var cost1 float64 = math.MaxFloat64
				if tier.DailyLimit >= dailyRequests {
					cost1 = tier.AnnualPriceRub / exchangeRate / 12.0
				}

				// Cost if we use this tier's license + overage (only applicable when overage is defined)
				var cost2 float64 = math.MaxFloat64
				if tier.OveragePerRub > 0 && dailyRequests > tier.DailyLimit {
					overageDaily := dailyRequests - tier.DailyLimit
					overageMonthly := overageDaily * daysInMonth
					overageRubMonthly := (float64(overageMonthly) / 1000.0) * tier.OveragePerRub
					// If cost1 is MaxFloat64 (invalid), cost2 will be the only valid option
					cost2 = (tier.AnnualPriceRub / exchangeRate / 12.0) + (overageRubMonthly / exchangeRate)
				}

				// Choose the cheaper valid option for this tier
				var tierCost float64 = math.MaxFloat64
				if cost2 < cost1 {
					tierCost = cost2
				} else {
					tierCost = cost1
				}

				// Only consider this tier if we have a valid cost (either a covering license or a license+overage)
				if tierCost < math.MaxFloat64 && tierCost < bestCost {
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

	// Add Yandex licence note if provider is yandex
	if strings.ToLower(providerID) == "yandex" {
		if notes != "" {
			notes = notes + "; "
		}
		notes = notes + "licence for year"
	}

	// compute per-request estimate
	totalRequests := 0
	for _, r := range apiRequests {
		totalRequests += r
	}
	var perRequest float64
	if totalRequests > 0 {
		perRequest = roundCost(totalCost / float64(totalRequests))
	}

	return domain.CalculationResult{
		Provider:   providerID,
		Name:       providerPricing.Name,
		URL:        providerPricing.URL,
		Cost:       roundCost(totalCost),
		PerRequest: perRequest,
		Breakdown:  breakdown,
		Notes:      notes,
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

	// Use 30 days in month for monthly <-> daily conversions
	daysInMonth := 30

	dailyRequestsNeeded := totalMonthlyRequests / daysInMonth
	if totalMonthlyRequests%daysInMonth > 0 {
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
		exchangeRate = fallbackFor("RUB")
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
			overageMonthly := overageDaily * daysInMonth
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

	if strings.ToLower(providerID) == "yandex" {
		notes = notes + "; licence for year"
	}

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
		Provider: providerID,
		Name:     providerPricing.Name,
		URL:      providerPricing.URL,
		Cost:     roundCost(bestCost),
		PerRequest: func() float64 {
			if totalMonthlyRequests > 0 {
				return roundCost(bestCost / float64(totalMonthlyRequests))
			}
			return 0
		}(),
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
	// Standard rounding to two decimal places (half up)
	return math.Round(cost*100.0) / 100.0
}

// roundCurrency rounds to 2 decimal places using standard rounding (half up)
func roundCurrency(val float64) float64 {
	return math.Round(val*100.0) / 100.0
}

func formatAPITypeName(apiType string) string {
	if name, exists := apiTypeNames[apiType]; exists {
		return name
	}
	return apiType
}
