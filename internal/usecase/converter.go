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
	apiKeyHeader = "X-API-Key"
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

		return 0.0, err
	}
	req.Header.Add(apiKeyHeader, c.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.log.Error("failed to send request", slog.Any("err", err))

		return 1, err
	}
	if resp.StatusCode != http.StatusOK {
		c.log.Debug("received not expected http states code", slog.Int("status_code", resp.StatusCode))
		return 1, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read response body", slog.Any("error", err))

		return 1, err
	}
	if len(body) == 0 {
		c.log.Debug("empty body, some error")

		return 1, nil
	}

	var result domain.CurrencyConvertResp
	if err := json.Unmarshal(body, &result); err != nil {
		c.log.Error("failed to unmarshall response", slog.Any("error", err))
	}

	c.log.Info("received exchange value", slog.Any("exchange_rate", result.Result.Rub))
	return result.Result.Rub, nil
}
