---
title: Grid Trading
category: trading_strategies
tags: [grid, range-trading, automation]
difficulty: beginner
---

# Grid Trading

## Concept
Place buy and sell orders at predetermined price intervals above and below a reference price. Profit from oscillations within a range.

## Grid Setup
1. **Reference price**: Current market price or moving average
2. **Grid spacing**: Based on ATR(14) or fixed percentage (e.g., 1-3%)
3. **Grid levels**: 5-10 levels each side
4. **Order size**: Equal size per level, or tapering (larger at extremes)

## Parameters
- **Upper/Lower bounds**: Define trading range
- **Grid count**: More grids = more trades, smaller profit per trade
- **Order size per grid**: Position size / grid count
- **Stop/reverse**: What happens when price leaves the range

## Best Markets for Grid Trading
- Ranging markets with clear support/resistance
- High volatility (frequent grid touches)
- Low transaction costs
- Crypto markets (24/7 trading, volatile ranges)

## Risk
- **Trending market**: Grid gets exhausted, left with underwater positions
- **Gap risk**: Price gaps through multiple levels
- **Capital intensive**: Need capital for all open grid levels
- **Mitigation**: Dynamic grid that shifts with trend, or trend filter to pause grid in strong trends

## A-Share Adaptation
- T+1 means grid orders execute one day at a time
- Daily price limits provide natural grid bounds
- Better suited for ETFs (lower commissions, no single-stock risk)
