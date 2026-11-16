package crypto

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PriceInfo struct {
	PriceUSD  float64   `json:"price_usd"`
	UpdatedAt time.Time `json:"updated_at"`
	Source    string    `json:"source"`
}

type PricingService struct {
	mu         sync.RWMutex
	cache      map[string]*PriceInfo
	httpClient *http.Client
	cacheTTL   time.Duration
}

var (
	defaultPricingService *PricingService
	pricingOnce           sync.Once
)

func GetPricingService() *PricingService {
	pricingOnce.Do(func() {
		defaultPricingService = NewPricingService(60 * time.Second)
	})
	return defaultPricingService
}

func NewPricingService(cacheTTL time.Duration) *PricingService {
	ps := &PricingService{
		cache: make(map[string]*PriceInfo),
		httpClient: &http.Client{
			Timeout: 6 * time.Second,
		},
		cacheTTL: cacheTTL,
	}
	// Seed initial baseline fallback rates
	now := time.Now().UTC()
	ps.cache["USDT"] = &PriceInfo{PriceUSD: 1.0, UpdatedAt: now, Source: "pegged"}
	ps.cache["USDC"] = &PriceInfo{PriceUSD: 1.0, UpdatedAt: now, Source: "pegged"}
	ps.cache["BTC"] = &PriceInfo{PriceUSD: 95000.0, UpdatedAt: now, Source: "seed"}
	ps.cache["ETH"] = &PriceInfo{PriceUSD: 3400.0, UpdatedAt: now, Source: "seed"}
	ps.cache["BNB"] = &PriceInfo{PriceUSD: 650.0, UpdatedAt: now, Source: "seed"}
	ps.cache["TRX"] = &PriceInfo{PriceUSD: 0.20, UpdatedAt: now, Source: "seed"}
	ps.cache["POL"] = &PriceInfo{PriceUSD: 0.50, UpdatedAt: now, Source: "seed"}
	ps.cache["SOL"] = &PriceInfo{PriceUSD: 210.0, UpdatedAt: now, Source: "seed"}
	ps.cache["LTC"] = &PriceInfo{PriceUSD: 95.0, UpdatedAt: now, Source: "seed"}
	ps.cache["DOGE"] = &PriceInfo{PriceUSD: 0.38, UpdatedAt: now, Source: "seed"}
	return ps
}

// GetPriceUSD returns the current price of a cryptocurrency in USD.
func (p *PricingService) GetPriceUSD(ctx context.Context, symbol string) (float64, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "USDT" || sym == "USDC" {
		return 1.0, nil
	}

	p.mu.RLock()
	cached, ok := p.cache[sym]
	if ok && time.Since(cached.UpdatedAt) < p.cacheTTL {
		val := cached.PriceUSD
		p.mu.RUnlock()
		return val, nil
	}
	p.mu.RUnlock()

	// Fetch live price
	livePrice, source, err := p.fetchLivePrice(ctx, sym)
	if err == nil && livePrice > 0 {
		p.mu.Lock()
		p.cache[sym] = &PriceInfo{
			PriceUSD:  livePrice,
			UpdatedAt: time.Now().UTC(),
			Source:    source,
		}
		p.mu.Unlock()
		return livePrice, nil
	}

	// Fallback to existing cached value if live fetch fails (rate limit / network)
	p.mu.RLock()
	if cached, ok := p.cache[sym]; ok && cached.PriceUSD > 0 {
		val := cached.PriceUSD
		p.mu.RUnlock()
		return val, nil
	}
	p.mu.RUnlock()

	return 0, fmt.Errorf("unable to determine price for %s: %w", sym, err)
}

func (p *PricingService) fetchLivePrice(ctx context.Context, symbol string) (float64, string, error) {
	// 1. Try Binance Public Ticker
	binanceSymbol := symbol + "USDT"
	binanceURL := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", binanceSymbol)
	req, _ := http.NewRequestWithContext(ctx, "GET", binanceURL, nil)
	resp, err := p.httpClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var res struct {
			Price string `json:"price"`
		}
		if decErr := json.NewDecoder(resp.Body).Decode(&res); decErr == nil {
			var price float64
			if _, sErr := fmt.Sscanf(res.Price, "%f", &price); sErr == nil && price > 0 {
				return price, "binance", nil
			}
		}
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	// 2. Try Coinbase Public Spot Price
	coinbaseURL := fmt.Sprintf("https://api.coinbase.com/v2/prices/%s-USD/spot", symbol)
	req2, _ := http.NewRequestWithContext(ctx, "GET", coinbaseURL, nil)
	resp2, err2 := p.httpClient.Do(req2)
	if err2 == nil && resp2.StatusCode == http.StatusOK {
		defer resp2.Body.Close()
		var res struct {
			Data struct {
				Amount string `json:"amount"`
			} `json:"data"`
		}
		if decErr := json.NewDecoder(resp2.Body).Decode(&res); decErr == nil {
			var price float64
			if _, sErr := fmt.Sscanf(res.Data.Amount, "%f", &price); sErr == nil && price > 0 {
				return price, "coinbase", nil
			}
		}
	}
	if resp2 != nil {
		_ = resp2.Body.Close()
	}

	return 0, "", fmt.Errorf("all price providers unavailable for %s", symbol)
}

// ConvertFiatToCrypto calculates the exact required crypto token amount for a given fiat USD value.
func (p *PricingService) ConvertFiatToCrypto(ctx context.Context, fiatAmountUSD float64, symbol string, decimals int) (float64, float64, error) {
	rate, err := p.GetPriceUSD(ctx, symbol)
	if err != nil || rate <= 0 {
		return 0, 0, err
	}

	rawAmount := fiatAmountUSD / rate

	// Precision formatting according to token decimals
	roundDecimals := decimals
	if roundDecimals > 8 {
		roundDecimals = 8 // Round display/invoice amount to 8 decimals max for usability
	}
	pow := math.Pow(10, float64(roundDecimals))
	cryptoAmount := math.Ceil(rawAmount*pow) / pow // Ceiling ensures client doesn't inadvertently underpay

	return cryptoAmount, rate, nil
}
