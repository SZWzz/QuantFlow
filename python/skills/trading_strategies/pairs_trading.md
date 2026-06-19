---
title: Pairs Trading
category: trading_strategies
tags: [pairs-trading, cointegration, statistical-arbitrage, mean-reversion]
difficulty: advanced
---

# Pairs Trading

## Strategy Overview
Find two highly correlated/cointegrated stocks. When their price ratio diverges from historical norm: short the overperformer, long the underperformer. Profit when the spread reverts.

## Pair Selection Criteria
1. **Same industry**: Both banks, both liquor, both EV makers
2. **High correlation**: >0.8 daily returns correlation over 1 year
3. **Cointegration**: Pass Engle-Granger test (p < 0.05)
4. **Liquidity**: Both stocks trade >10M CNY daily
5. **Fundamental similarity**: Similar market cap, business model

## Entry/Exit Rules
- **Entry**: Spread > 2 standard deviations from mean
- **Exit**: Spread reverts to within 0.5 standard deviations
- **Stop loss**: Spread reaches 3 standard deviations
- **Time stop**: 20 trading days maximum holding

## Position Sizing
- Dollar-neutral: equal capital long and short
- Beta-neutral: adjust sizes for different betas
- Notional: same number of shares each side (simpler but less precise)

## A-Share Specifics
- **Short constraints**: A-share short selling requires margin account and borrow availability
- **Alternative**: Use index futures (IF/IC/IH) as the short leg, stock as long leg
- **Cost**: Borrow fee (~2-8% annualized) + margin interest
- **Pair candidates**: Ping An vs China Life (insurers), Kweichow Moutai vs Wuliangye (liquor)
