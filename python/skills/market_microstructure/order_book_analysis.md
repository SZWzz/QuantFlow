---
title: Order Book Analysis
category: market_microstructure
tags: [order-book, depth, spread, liquidity]
difficulty: advanced
---

# Order Book Analysis

## Key Metrics

### Bid-Ask Spread
- **Quoted spread**: ask - bid
- **Effective spread**: 2 * |trade_price - mid_price|
- **Realized spread**: 2 * |trade_price - mid_price_future|
- Wider spreads = higher trading costs

### Market Depth
- Total volume available at each price level
- Depth imbalance: (bid_volume - ask_volume) / (bid_volume + ask_volume)
- Positive imbalance = buying pressure, predicts short-term price increase

### Order Flow Imbalance (OFI)
- Change in bid/ask quantities between snapshots
- OFI = Δbid_qty - Δask_qty
- Strong predictor of short-term price moves (1-10 seconds)

## Market Making Signals
- **Inventory**: High long inventory → lower bid, reduce buying
- **Adverse selection**: Informed traders cause losses; need wider spreads
- **Volatility**: Higher vol → wider spreads

## A-Share Level-2 Data
- Available from EastMoney, Futu, broker terminals
- Top 5 bid/ask levels standard, top 10 available
- Tick-level data: ~3 seconds per snapshot
