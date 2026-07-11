package research

import (
	"math"
)

// OptionType represents call or put.
type OptionType string

const (
	Call OptionType = "call"
	Put  OptionType = "put"
)

// OptionGreeks holds the standard option Greeks.
type OptionGreeks struct {
	Delta float64 `json:"delta"`
	Gamma float64 `json:"gamma"`
	Theta float64 `json:"theta"` // per day
	Vega  float64 `json:"vega"`  // per 1% vol change
	Rho   float64 `json:"rho"`   // per 1% rate change
}

// OptionPrice computes Black-Scholes price and Greeks for European options.
// S=spot, K=strike, T=years to expiry, r=risk-free rate, sigma=volatility.
func OptionPrice(optType OptionType, S, K, T, r, sigma float64) (price float64, greeks OptionGreeks) {
	if T <= 0 || sigma <= 0 || S <= 0 {
		if optType == Call {
			price = math.Max(0, S-K)
		} else {
			price = math.Max(0, K-S)
		}
		return
	}

	d1 := (math.Log(S/K) + (r+sigma*sigma/2)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)

	nd1 := normalCDF(d1)
	nd2 := normalCDF(d2)
	nmd1 := normalCDF(-d1)
	nmd2 := normalCDF(-d2)
	npd1 := normalPDF(d1)

	discount := math.Exp(-r * T)

	if optType == Call {
		price = S*nd1 - K*discount*nd2
		greeks.Delta = nd1
		greeks.Theta = (-S*npd1*sigma/(2*math.Sqrt(T)) - r*K*discount*nd2) / 365
		greeks.Rho = K * T * discount * nd2 / 100
	} else {
		price = K*discount*nmd2 - S*nmd1
		greeks.Delta = -nmd1
		greeks.Theta = (-S*npd1*sigma/(2*math.Sqrt(T)) + r*K*discount*nmd2) / 365
		greeks.Rho = -K * T * discount * nmd2 / 100
	}

	greeks.Gamma = npd1 / (S * sigma * math.Sqrt(T))
	greeks.Vega = S * npd1 * math.Sqrt(T) / 100

	return
}

// ImpliedVolatility computes implied volatility from option market price.
// Uses Newton-Raphson iteration.
func ImpliedVolatility(optType OptionType, marketPrice, S, K, T, r float64) float64 {
	if marketPrice <= 0 || T <= 0 {
		return 0
	}

	// Initial guess
	sigma := 0.30
	for i := 0; i < 100; i++ {
		price, greeks := OptionPrice(optType, S, K, T, r, sigma)
		vega := greeks.Vega * 100 // vega is per 1%, convert to per 100%

		diff := price - marketPrice
		if math.Abs(diff) < 1e-6 {
			return sigma
		}

		if vega <= 1e-10 {
			// Vega near zero, switch to bisection
			break
		}

		sigma -= diff / vega

		if sigma <= 0.001 {
			sigma = 0.001
		}
		if sigma > 5.0 {
			sigma = 5.0
		}
	}

	// Fallback: bisection
	lo, hi := 0.001, 5.0
	for i := 0; i < 50; i++ {
		mid := (lo + hi) / 2
		price, _ := OptionPrice(optType, S, K, T, r, mid)
		if price > marketPrice {
			hi = mid
		} else {
			lo = mid
		}
		if hi-lo < 1e-6 {
			return mid
		}
	}
	return (lo + hi) / 2
}

// BinomialPrice computes option price using CRR binomial tree.
// Supports both European and American options.
func BinomialPrice(optType OptionType, S, K, T, r, sigma float64, steps int, american bool) float64 {
	if steps <= 0 {
		steps = 100
	}
	if T <= 0 {
		if optType == Call {
			return math.Max(0, S-K)
		}
		return math.Max(0, K-S)
	}

	dt := T / float64(steps)
	u := math.Exp(sigma * math.Sqrt(dt))
	d := 1 / u
	p := (math.Exp(r*dt) - d) / (u - d)
	discount := math.Exp(-r * dt)

	// Initialize terminal prices
	prices := make([]float64, steps+1)
	for j := 0; j <= steps; j++ {
		st := S * math.Pow(u, float64(steps-j)) * math.Pow(d, float64(j))
		if optType == Call {
			prices[j] = math.Max(0, st-K)
		} else {
			prices[j] = math.Max(0, K-st)
		}
	}

	// Backward induction
	for i := steps - 1; i >= 0; i-- {
		for j := 0; j <= i; j++ {
			hold := discount * (p*prices[j] + (1-p)*prices[j+1])
			if american {
				st := S * math.Pow(u, float64(i-j)) * math.Pow(d, float64(j))
				var exercise float64
				if optType == Call {
					exercise = math.Max(0, st-K)
				} else {
					exercise = math.Max(0, K-st)
				}
				prices[j] = math.Max(hold, exercise)
			} else {
				prices[j] = hold
			}
		}
	}

	return prices[0]
}

// normalPDF is the standard normal probability density function.
func normalPDF(x float64) float64 {
	return math.Exp(-x*x/2) / math.Sqrt(2*math.Pi)
}

