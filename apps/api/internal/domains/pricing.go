package domains

import (
	"context"
	"fmt"
	"math"
	"strings"

	"hostvra/api/internal/store"
)

type PricingEngine struct {
	store store.Store
}

func NewPricingEngine(s store.Store) *PricingEngine {
	return &PricingEngine{store: s}
}

type CalculatedPrice struct {
	TLD               string  `json:"tld"`
	Years             int     `json:"years"`
	UnitPrice         float64 `json:"unit_price"`
	TotalPrice        float64 `json:"total_price"`
	Currency          string  `json:"currency"`
	WholesaleUnitCost float64 `json:"-"` // Internal only
	WholesaleTotalCost float64 `json:"-"` // Internal only
}

// CalculateRegistrationPrice returns customer selling price and internal cost
func (pe *PricingEngine) CalculateRegistrationPrice(ctx context.Context, tld string, years int) (*CalculatedPrice, error) {
	if years < 1 {
		years = 1
	}
	cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")

	price, err := pe.store.GetDomainPrice(ctx, cleanTLD)
	if err != nil {
		return nil, fmt.Errorf("pricing not configured for TLD: %s", cleanTLD)
	}

	if !price.Enabled {
		return nil, fmt.Errorf("registration is currently disabled for TLD: %s", cleanTLD)
	}

	unitPrice := price.RegistrationPrice
	totalPrice := roundMoney(unitPrice * float64(years))
	totalCost := roundMoney(price.RegistrationCost * float64(years))

	return &CalculatedPrice{
		TLD:                cleanTLD,
		Years:              years,
		UnitPrice:          unitPrice,
		TotalPrice:         totalPrice,
		Currency:           price.Currency,
		WholesaleUnitCost:  price.RegistrationCost,
		WholesaleTotalCost: totalCost,
	}, nil
}

// CalculateRenewalPrice returns renewal price for customer
func (pe *PricingEngine) CalculateRenewalPrice(ctx context.Context, tld string, years int) (*CalculatedPrice, error) {
	if years < 1 {
		years = 1
	}
	cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")

	price, err := pe.store.GetDomainPrice(ctx, cleanTLD)
	if err != nil {
		return nil, fmt.Errorf("pricing not configured for TLD: %s", cleanTLD)
	}

	unitPrice := price.RenewalPrice
	totalPrice := roundMoney(unitPrice * float64(years))
	totalCost := roundMoney(price.RenewalCost * float64(years))

	return &CalculatedPrice{
		TLD:                cleanTLD,
		Years:              years,
		UnitPrice:          unitPrice,
		TotalPrice:         totalPrice,
		Currency:           price.Currency,
		WholesaleUnitCost:  price.RenewalCost,
		WholesaleTotalCost: totalCost,
	}, nil
}

// CalculateTransferPrice returns transfer price for customer
func (pe *PricingEngine) CalculateTransferPrice(ctx context.Context, tld string) (*CalculatedPrice, error) {
	cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")

	price, err := pe.store.GetDomainPrice(ctx, cleanTLD)
	if err != nil {
		return nil, fmt.Errorf("pricing not configured for TLD: %s", cleanTLD)
	}

	return &CalculatedPrice{
		TLD:                cleanTLD,
		Years:              1,
		UnitPrice:          price.TransferPrice,
		TotalPrice:         price.TransferPrice,
		Currency:           price.Currency,
		WholesaleUnitCost:  price.TransferCost,
		WholesaleTotalCost: price.TransferCost,
	}, nil
}

func roundMoney(val float64) float64 {
	return math.Round(val*100) / 100
}
