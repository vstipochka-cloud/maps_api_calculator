package domain

type CalculationRequest struct {
	APIRequests              map[string]int `json:"api_requests"`
	DisableNewCustomerCredit bool           `json:"disable_new_customer_credit"`
	DisableFreeTier          bool           `json:"disable_free_tier"`
	Currency                 string         `json:"currency"`
}

type PricingData struct {
	Metadata struct {
		Currency    string `json:"currency"`
		LastUpdated string `json:"last_updated"`
		Note        string `json:"note"`
	} `json:"metadata"`
	APITypes  []string                   `json:"api_types"`
	Providers map[string]ProviderPricing `json:"providers"`
}

type ProviderPricing struct {
	Name            string                `json:"name"`
	URL             string                `json:"url"`
	FreeCredit      *FreeCreditInfo       `json:"free_credit"`
	MonthlyFreeTier *MonthlyFreeTierInfo  `json:"monthly_free_tier"`
	APIs            map[string]APIPricing `json:"apis"`
}

type FreeCreditInfo struct {
	Type           string  `json:"type"` // "one_time" or "monthly"
	AmountUSD      float64 `json:"amount_usd"`
	AppliesTo      string  `json:"applies_to"` // "new_accounts_only" or "all"
	DurationMonths int     `json:"duration_months"`
	Note           string  `json:"note"`
}

type MonthlyFreeTierInfo struct {
	Type           string `json:"type"`            // "shared_pool" or "per_api"
	AmountRequests int    `json:"amount_requests"` // shared pool total
	Note           string `json:"note"`
}

type APIPricing struct {
	Unit         string        `json:"unit"`
	PricePer1000 float64       `json:"price_per_1000,omitempty"` // deprecated, use Tiers
	Tiers        []PricingTier `json:"tiers,omitempty"`          // NEW: volume-based pricing
	FreeTier     int           `json:"free_tier"`
}

// PricingTier represents a volume-based pricing level
type PricingTier struct {
	FromRequests int     `json:"from_requests"` // Start of tier (inclusive)
	ToRequests   int     `json:"to_requests"`   // End of tier (inclusive), 0 = unlimited
	PricePer1000 float64 `json:"price_per_1000"`
}

type CalculationResult struct {
	Provider      string                      `json:"provider"`
	Name          string                      `json:"name"`
	URL           string                      `json:"url"`
	Cost          float64                     `json:"cost"`
	ConvertedCost float64                     `json:"converted_cost"`
	Breakdown     map[string]APICostBreakdown `json:"breakdown"`
	Notes         string                      `json:"notes,omitempty"`
}

type APICostBreakdown struct {
	Requests       int     `json:"requests"`
	UnitPrice      float64 `json:"unit_price"`
	FreeTier       int     `json:"free_tier"`
	BilledRequests int     `json:"billed_requests"`
	Cost           float64 `json:"cost"`
	ConvertedCost  float64 `json:"converted_cost"`
}

type CalculationResponse struct {
	Results       []CalculationResult `json:"results"`
	BestValue     string              `json:"best_value"`
	TotalCost     float64             `json:"total_cost"`
	BaseCurrency  string              `json:"base_currency"`
	Currency      string              `json:"currency"`
	ExchangeRate  float64             `json:"exchange_rate"`
	ConvertedCost float64             `json:"converted_cost"`
}

type CurrencyConvertResp struct {
	Result CurrencyResult `json:"result"`
}

type CurrencyResult struct {
	Rub float64 `json:"RUB"`
}
