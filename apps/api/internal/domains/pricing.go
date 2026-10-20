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

var defaultCatalogTLDPrices = map[string]struct {
	RegPrice   float64
	RenewPrice float64
	TransPrice float64
	Cost       float64
}{
	"com":    {RegPrice: 14.99, RenewPrice: 16.99, TransPrice: 14.99, Cost: 10.29},
	"net":    {RegPrice: 16.99, RenewPrice: 18.99, TransPrice: 16.99, Cost: 12.49},
	"org":    {RegPrice: 15.99, RenewPrice: 17.99, TransPrice: 15.99, Cost: 11.89},
	"xyz":    {RegPrice: 2.99,  RenewPrice: 13.99, TransPrice: 12.99, Cost: 1.99},
	"io":     {RegPrice: 44.99, RenewPrice: 49.99, TransPrice: 44.99, Cost: 32.50},
	"co":     {RegPrice: 14.99, RenewPrice: 29.99, TransPrice: 27.99, Cost: 9.99},
	"tech":   {RegPrice: 5.99,  RenewPrice: 23.99, TransPrice: 21.99, Cost: 3.89},
	"store":  {RegPrice: 4.99,  RenewPrice: 34.99, TransPrice: 31.99, Cost: 2.99},
	"online": {RegPrice: 2.99,  RenewPrice: 28.99, TransPrice: 26.99, Cost: 1.89},
	"info":   {RegPrice: 6.99,  RenewPrice: 21.99, TransPrice: 19.99, Cost: 4.29},
	"biz":    {RegPrice: 11.99, RenewPrice: 21.99, TransPrice: 19.99, Cost: 7.50},
}

// CalculateRegistrationPrice returns customer selling price and internal cost
func (pe *PricingEngine) CalculateRegistrationPrice(ctx context.Context, tld string, years int) (*CalculatedPrice, error) {
	if years < 1 {
		years = 1
	}
	cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")

	price, err := pe.store.GetDomainPrice(ctx, cleanTLD)
	if err != nil || price == nil {
		if fb, ok := defaultCatalogTLDPrices[cleanTLD]; ok {
			unitPrice := fb.RegPrice
			return &CalculatedPrice{
				TLD:                cleanTLD,
				Years:              years,
				UnitPrice:          unitPrice,
				TotalPrice:         roundMoney(unitPrice * float64(years)),
				Currency:           "USD",
				WholesaleUnitCost:  fb.Cost,
				WholesaleTotalCost: roundMoney(fb.Cost * float64(years)),
			}, nil
		}
		// Generic default for unlisted TLD
		return &CalculatedPrice{
			TLD:                cleanTLD,
			Years:              years,
			UnitPrice:          14.99,
			TotalPrice:         roundMoney(14.99 * float64(years)),
			Currency:           "USD",
			WholesaleUnitCost:  10.00,
			WholesaleTotalCost: roundMoney(10.00 * float64(years)),
		}, nil
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
	if err != nil || price == nil {
		renewPrice := 16.99
		cost := 12.00
		if fb, ok := defaultCatalogTLDPrices[cleanTLD]; ok {
			renewPrice = fb.RenewPrice
			cost = fb.Cost
		}
		return &CalculatedPrice{
			TLD:                cleanTLD,
			Years:              years,
			UnitPrice:          renewPrice,
			TotalPrice:         roundMoney(renewPrice * float64(years)),
			Currency:           "USD",
			WholesaleUnitCost:  cost,
			WholesaleTotalCost: roundMoney(cost * float64(years)),
		}, nil
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
	if err != nil || price == nil {
		transPrice := 14.99
		cost := 10.00
		if fb, ok := defaultCatalogTLDPrices[cleanTLD]; ok {
			transPrice = fb.TransPrice
			cost = fb.Cost
		}
		return &CalculatedPrice{
			TLD:                cleanTLD,
			Years:              1,
			UnitPrice:          transPrice,
			TotalPrice:         transPrice,
			Currency:           "USD",
			WholesaleUnitCost:  cost,
			WholesaleTotalCost: cost,
		}, nil
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
