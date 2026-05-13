package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const (
	apiKeyHeader              = "X-API-Key"
	fallbackExchangeRateRUB   = 74.0 // Fallback exchange rate for RUB/USD (requested)
	fallbackExchangeRateEUR   = 0.85 // Fallback exchange rate for EUR/USD (requested)
	fallbackExchangeRateOther = 1.0  // Generic fallback
)

type Converter interface {
	Convert(
		ctx context.Context,
		currency string,
	) (float64, error)
}

type CurrencyConverter struct {
	log         slog.Logger
	providerUrl string
	apiKey      string
}

func NewCurrencyConverter(providerUrl, apiKey string, log slog.Logger) *CurrencyConverter {
	return &CurrencyConverter{
		log:         log,
		providerUrl: providerUrl,
		apiKey:      apiKey,
	}
}

func (c *CurrencyConverter) Convert(
	ctx context.Context, currency string,
) (float64, error) {
	if currency == "" || currency == "USD" {
		return 1, nil
	}

	client := &http.Client{}

	// Build request for exchangerate-api style: {providerUrl}/{apiKey}/pair/USD/{TARGET}
	target := strings.ToUpper(currency)
	base := "USD"
	url := fmt.Sprintf("%s/%s/pair/%s/%s", strings.TrimRight(c.providerUrl, "/"), c.apiKey, base, target)
	c.log.Info("BBBBB", slog.String("url", url))
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		url,
		nil,
	)
	if err != nil {
		c.log.Error("failed to create request", slog.Any("err", err))

		return fallbackFor(currency), err
	}
	req.Header.Add(apiKeyHeader, c.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.log.Error("failed to send request", slog.Any("err", err))

		return fallbackFor(currency), err
	}
	if resp.StatusCode != http.StatusOK {
		c.log.Debug("received not expected http status code, using fallback exchange rate", slog.Int("status_code", resp.StatusCode), slog.String("currency", currency), slog.Float64("fallback_rate", fallbackFor(currency)))
		return fallbackFor(currency), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read response body", slog.Any("error", err))

		return fallbackFor(currency), err
	}
	if len(body) == 0 {
		c.log.Debug("empty body, using fallback exchange rate", slog.String("currency", currency), slog.Float64("fallback_rate", fallbackFor(currency)))

		return fallbackFor(currency), nil
	}

	// Try parsing exchangerate-api pair response first
	var pairResp struct {
		ConversionRate float64 `json:"conversion_rate"`
		BaseCode       string  `json:"base_code"`
		TargetCode     string  `json:"target_code"`
		Result         string  `json:"result"`
	}

	if err := json.Unmarshal(body, &pairResp); err == nil {
		if pairResp.ConversionRate > 0 {
			c.log.Info("received exchange value", slog.String("currency", target), slog.Float64("exchange_rate", pairResp.ConversionRate))
			return pairResp.ConversionRate, nil
		}
	}

	c.log.Debug("no exchange rate found in response body, using configured fallback", slog.String("currency", currency), slog.Float64("fallback_rate", fallbackFor(currency)))
	return fallbackFor(currency), nil
}

func fallbackFor(currency string) float64 {
	switch strings.ToUpper(currency) {
	case "RUB":
		return fallbackExchangeRateRUB
	case "EUR":
		return fallbackExchangeRateEUR
	default:
		return fallbackExchangeRateOther
	}
}
