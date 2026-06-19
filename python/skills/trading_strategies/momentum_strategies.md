---
title: Momentum Strategies
category: trading_strategies
tags: [momentum, trend-following, factor]
difficulty: intermediate
---

# Momentum Strategies

## Core Concept
Momentum is the tendency of assets that have performed well (poorly) in the recent past to continue performing well (poorly) in the near future.

## Types

### Cross-Sectional Momentum
Buy past winners, sell past losers within a defined universe.
- Typical look-back: 3, 6, or 12 months (skip most recent month to avoid short-term reversal)
- Rebalance: monthly
- Universe: top 80% by market cap or liquidity

### Time-Series Momentum
Go long assets with positive recent excess returns, short those with negative.
- Look-back: 1, 3, 6, 12 months
- Volatility scaling: position size = target_vol / realized_vol
- Often applied to futures and macro assets

## Implementation Checklist

1. **Universe construction**: Filter by liquidity (avg daily volume > 10M for A-shares)
2. **Signal generation**: Compute N-month return, skip T-1 month
3. **Portfolio construction**: Equal-weight or score-weighted, top quintile long, bottom quintile short
4. **Rebalance**: Month-end, T+1 effective for A-shares
5. **Risk management**: Sector neutrality, max position 5%, stop-loss at -2 STD

## A-Share Specifics
- T+1 settlement means signals generated on day T take effect on day T+1
- Price limits (±10% for most stocks) may prevent entry/exit at signal price
- Stamp duty (0.05% on sell) increases turnover cost; prefer longer holding periods
- CSI 300/500/1000 for broad universe coverage

## Evaluation Metrics
- Sharpe ratio (>1.0 is good)
- Max drawdown (<25% is acceptable)
- Turnover (<100% monthly is reasonable)
- Factor IC (information coefficient) and ICIR

## Common Pitfalls
- Look-ahead bias: ensure signal uses only data available at decision time
- Survivorship bias: include delisted stocks in historical universe
- Micro-cap noise: filter out stocks with market cap < 1B CNY
