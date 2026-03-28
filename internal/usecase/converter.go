package usecase

import (
	"calculator_api/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const (
	apiKeyHeader            = "X-API-Key"
	fallbackExchangeRateRUB = 95.0 // Fallback exchange rate for RUB/USD
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

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/fetch-one?from=USD&to=%s", c.providerUrl, currency),
		nil,
	)
	if err != nil {
		c.log.Error("failed to create request", slog.Any("err", err))

		return fallbackExchangeRateRUB, err
	}
	req.Header.Add(apiKeyHeader, c.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.log.Error("failed to send request", slog.Any("err", err))

		return fallbackExchangeRateRUB, err
	}
	if resp.StatusCode != http.StatusOK {
		c.log.Debug("received not expected http states code, using fallback exchange rate", slog.Int("status_code", resp.StatusCode), slog.Float64("fallback_rate", fallbackExchangeRateRUB))
		return fallbackExchangeRateRUB, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read response body", slog.Any("error", err))

		return fallbackExchangeRateRUB, err
	}
	if len(body) == 0 {
		c.log.Debug("empty body, using fallback exchange rate")

		return fallbackExchangeRateRUB, nil
	}

	var result domain.CurrencyConvertResp
	if err := json.Unmarshal(body, &result); err != nil {
		c.log.Error("failed to unmarshall response", slog.Any("error", err))
		return fallbackExchangeRateRUB, err
	}

	c.log.Info("received exchange value", slog.Any("exchange_rate", result.Result.Rub))
	return result.Result.Rub, nil
}
