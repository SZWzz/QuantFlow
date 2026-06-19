---
title: Market Impact Models
category: market_microstructure
tags: [market-impact, execution, algo-trading]
difficulty: advanced
---

# Market Impact Models

## Components of Trading Cost
1. **Commissions + fees**: Explicit, known ex-ante
2. **Bid-ask spread**: Half-spread per trade
3. **Market impact**: Price moves against your order
4. **Delay cost**: Price moves while waiting to execute
5. **Opportunity cost**: Cost of not completing the order

## Almgren-Chriss Model
- Temporary impact: decays after trade (liquidity effect)
- Permanent impact: persistent price change (information effect)
- Optimal schedule balances impact vs timing risk

## Square Root Law
- Impact ≈ σ * sqrt(Q / V)
- σ = daily volatility
- Q = order size
- V = daily volume
- Widely observed empirically across markets

## Practical Rules of Thumb
- Trade ≤5% of daily volume to keep impact minimal
- Participation rate: 20-30% of market volume
- Larger orders → longer execution horizon
- Dark pools / block trading for orders >1% ADV

## A-Share Execution
- T+1 settlement: plan for next-day availability
- Price limits: orders beyond daily limit range are rejected
- 100-share lots: round down to lot multiple
