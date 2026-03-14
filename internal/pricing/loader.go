package pricing

import (
	"encoding/json"
	"os"
	"path/filepath"

	"calculator_api/internal/domain"
)

type Loader interface {
	Load(filePath string) (*domain.PricingData, error)
}

type JSONLoader struct{}

func NewJSONLoader() Loader {
	return &JSONLoader{}
}

func (l *JSONLoader) Load(filePath string) (*domain.PricingData, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	var pricing domain.PricingData
	if err := json.Unmarshal(data, &pricing); err != nil {
		return nil, err
	}

	return &pricing, nil
}
