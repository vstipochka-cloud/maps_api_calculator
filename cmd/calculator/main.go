package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"calculator_api/internal/domain"
	"calculator_api/internal/handler"
	"calculator_api/internal/pricing"
	"calculator_api/internal/usecase"

	"github.com/joho/godotenv"
)

const providerURL = "https://api.fastforex.io"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("Failed to get current working directory", "error", err)
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	slog.Info("Application starting", "cwd", cwd)

	if err := godotenv.Load(); err != nil {
		log.Fatal("failed to load env file")
	}

	pricingPaths := []string{
		filepath.Join(cwd, "cmd/calculator/pricing/pricing.json"),
		"cmd/calculator/pricing/pricing.json",
		"pricing/pricing.json",
		filepath.Join(cwd, "pricing/pricing.json"),
	}

	var pricingData *domain.PricingData
	var loadErr error
	var usedPath string

	loader := pricing.NewJSONLoader()

	for _, path := range pricingPaths {
		slog.Debug("Attempting to load pricing from", "path", path)
		pricingData, loadErr = loader.Load(path)
		if loadErr == nil {
			usedPath = path
			break
		}
		slog.Debug("Failed to load from path", "path", path, "error", loadErr)
	}

	if loadErr != nil {
		slog.Error("Failed to load pricing data from any location", "error", loadErr, "attempts", len(pricingPaths))
		log.Fatalf("Failed to load pricing data from any location: %v", loadErr)
	}

	slog.Info("Pricing data loaded successfully", "path", usedPath, "providers", len(pricingData.Providers))
	for providerID, provider := range pricingData.Providers {
		slog.Debug("Loaded provider", "provider_id", providerID, "name", provider.Name, "apis_count", len(provider.APIs))
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatalf("empty api key")
	}

	converter := usecase.NewCurrencyConverter(providerURL, apiKey, *logger)
	calcHandler := handler.NewCalculatorHandler(*converter, pricingData)

	// Setup middleware wrapper for CORS
	withCORS := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	// API endpoints only (no frontend serving)
	http.HandleFunc("/health", handler.LoggingMiddleware("health", withCORS(calcHandler.HealthHandler)))
	http.HandleFunc("/providers", handler.LoggingMiddleware("providers", withCORS(calcHandler.ProvidersHandler)))
	http.HandleFunc("/calculate", handler.LoggingMiddleware("calculate", withCORS(calcHandler.CalculateHandler)))

	port := ":8080"
	fmt.Printf("Starting calculator API server on http://localhost:8080\n")
	fmt.Printf("Available endpoints:\n")
	fmt.Printf("  GET  /health      - Check server status\n")
	fmt.Printf("  GET  /providers   - List available providers and API types\n")
	fmt.Printf("  POST /calculate   - Calculate API costs\n\n")
	fmt.Printf("Frontend: Run 'cd frontend && npm start' (or use live server)\n\n")

	slog.Info("Server starting", "port", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("Server failed to start", "error", err)
		log.Fatalf("Server failed to start: %v", err)
	}
}
