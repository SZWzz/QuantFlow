---
title: Position Sizing Methods
category: risk_management
tags: [position-sizing, kelly, risk-parity, allocation]
difficulty: intermediate
---

# Position Sizing Methods

## Equal Weight (1/N)
- Simplest: allocate equally across N positions
- Benchmark for more complex methods
- Works well when: high uncertainty about future returns

## Risk Parity
- Allocate such that each position contributes equal risk
- Need covariance matrix estimation
- More stable than mean-variance optimization

## Kelly Criterion
- Optimal fraction = edge / odds
- f* = (p * b - q) / b
  - p = win probability, q = 1-p, b = win/loss ratio
- Practical: use half-Kelly to reduce volatility

## Volatility Targeting
- Position size = target_daily_vol / asset_daily_vol
- Target: 1-2% daily vol per position
- Scales automatically with market conditions

## Fixed Fractional
- Risk fixed % of capital per trade (e.g., 1-2%)
- Position size = (capital * risk_pct) / stop_distance
- Most common among professional traders

## A-Share Constraints
- 100-share lot minimum (整手)
- 100-share increments above minimum
- Position limits: 5% of outstanding shares for significant shareholders
