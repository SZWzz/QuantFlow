---
title: Mean Reversion Strategies
category: trading_strategies
tags: [mean-reversion, pairs, bollinger, statistical-arbitrage]
difficulty: intermediate
---

# Mean Reversion Strategies

## Core Concept
Prices deviate from their mean temporarily and revert. Profit from the reversion.

## Bollinger Band Mean Reversion
- Buy when price touches lower band (2σ below 20-SMA)
- Sell when price touches upper band (2σ above 20-SMA)
- Works best in: ranging/sideways markets (low ADX <25)
- Fails in: strong trending markets (price "walks the band")

## RSI Extremes
- Buy: RSI(14) < 30 and starting to turn up
- Sell: RSI(14) > 70 and starting to turn down
- Add filter: only trade in direction of longer-term trend (200-SMA)

## Statistical Arbitrage
- Identify cointegrated pairs (Engle-Granger, Johansen tests)
- Enter when spread > 2σ from mean
- Exit when spread reverts to mean
- Half-life of mean reversion: determine holding period

## Risk Management
- Mean reversion can become momentum: always use stops
- Stop loss: 2x the entry deviation (e.g., if enter at 2σ, stop at 3σ)
- Position size: smaller than trend-following (mean reversion has lower win rate)
- Avoid during: earnings announcements, macro events, regime changes

## A-Share Adaptation
- Shanghai/Shenzhen have different sector compositions; analyze separately
- A/H premium mean reversion for dual-listed stocks
- Sector rotation: overbought sectors → oversold sectors
