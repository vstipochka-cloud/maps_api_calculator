package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"calculator_api/internal/domain"
	"calculator_api/internal/usecase"
)

type CalculatorHandler struct {
	calc        usecase.Calculator
	pricingData *domain.PricingData
}

func NewCalculatorHandler(converter usecase.CurrencyConverter, pricing *domain.PricingData) *CalculatorHandler {
	return &CalculatorHandler{
		calc:        usecase.NewCalculator(converter),
		pricingData: pricing,
	}
}

func (h *CalculatorHandler) CalculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		slog.Warn("Invalid method for /calculate", "method", r.Method)
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request body", "error", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	slog.Debug("Calculation request decoded", "api_requests_count", len(req.APIRequests), "disable_credit", req.DisableNewCustomerCredit)

	if len(req.APIRequests) == 0 {
		slog.Warn("Empty api_requests in calculation request")
		http.Error(w, "api_requests field is required and cannot be empty", http.StatusBadRequest)
		return
	}

	validAPITypes := make(map[string]bool)
	for _, apiType := range h.pricingData.APITypes {
		validAPITypes[apiType] = true
	}

	for apiType := range req.APIRequests {
		if !validAPITypes[apiType] {
			slog.Warn("Unknown API type requested", "api_type", apiType)
			http.Error(w, "Unknown API type: "+apiType, http.StatusBadRequest)
			return
		}
	}

	slog.Debug("Validating API types passed", "api_types_count", len(req.APIRequests))

	// Validate matrix_params if distance_matrix is requested
	if _, hasDistanceMatrix := req.APIRequests["distance_matrix"]; hasDistanceMatrix {
		if req.MatrixParams == nil || req.MatrixParams.OriginsCount == 0 || req.MatrixParams.DestinationsCount == 0 {
			slog.Warn("distance_matrix requested without proper matrix_params")
			http.Error(w, "distance_matrix requires matrix_params with origins_count > 0 and destinations_count > 0", http.StatusBadRequest)
			return
		}
		slog.Debug("Distance matrix params validated", "origins", req.MatrixParams.OriginsCount, "destinations", req.MatrixParams.DestinationsCount)
	}

	response := h.calc.Calculate(&req, h.pricingData)

	slog.Info("Calculation completed successfully",
		"best_value", response.BestValue,
		"total_cost", response.TotalCost,
		"providers_count", len(response.Results),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *CalculatorHandler) ProvidersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		slog.Warn("Invalid method for /providers", "method", r.Method)
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Debug("Providers information requested", "providers_count", len(h.pricingData.Providers))

	providers := make([]map[string]interface{}, 0, len(h.pricingData.Providers))

	for id, provider := range h.pricingData.Providers {
		p := map[string]interface{}{
			"id":   id,
			"name": provider.Name,
			"url":  provider.URL,
			"apis": getProviderAPIs(provider),
		}
		providers = append(providers, p)
		slog.Debug("Provider info prepared", "provider_id", id, "apis_count", len(provider.APIs))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": providers,
		"api_types": h.pricingData.APITypes,
	})

	slog.Info("Providers list sent", "providers_count", len(providers))
}

func (h *CalculatorHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Health check requested")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
	slog.Debug("Health check response sent")
}

func getProviderAPIs(provider domain.ProviderPricing) []string {
	apis := make([]string, 0, len(provider.APIs))
	for apiType := range provider.APIs {
		apis = append(apis, apiType)
	}
	return apis
}

func LoggingMiddleware(name string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Request received", "endpoint", name, "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		handler(w, r)
		slog.Info("Request completed", "endpoint", name, "method", r.Method)
	}
}
