---
title: Value at Risk (VaR) Methods
category: risk_management
tags: [var, cvar, risk-metrics, stress-test]
difficulty: advanced
---

# Value at Risk (VaR)

## Methods

### Historical VaR
- Uses actual historical returns distribution
- VaR(95%) = 5th percentile of historical returns
- Advantage: No distribution assumption
- Disadvantage: Assumes history repeats

### Parametric (Variance-Covariance) VaR
- Assumes normal distribution
- VaR(95%) = portfolio_value * (mean_return - 1.645 * std_return)
- Fast computation, but underestimates tail risk

### Monte Carlo VaR
- Simulate thousands of scenarios
- Most flexible: can model complex instruments
- Computationally intensive

## CVaR (Conditional VaR / Expected Shortfall)
- Average loss BEYOND VaR threshold
- Better captures tail risk than VaR
- Recommended by Basel III

## Stress Testing
- Historical scenarios: 2008 crisis, 2015 A-share crash, 2020 COVID
- Hypothetical: -30% market, +200bp rates, correlation breakdown

## A-Share Tail Risk
- Daily price limits create truncated distribution
- Historical VaR naturally accounts for limits
- Parametric VaR should be adjusted for kurtosis (fat tails)
