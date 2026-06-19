---
title: Portfolio Optimization
category: risk_management
tags: [optimization, markowitz, black-litterman, constraints]
difficulty: advanced
---

# Portfolio Optimization

## Mean-Variance Optimization (Markowitz)
- Minimize: w'Σw - λ * w'μ
- Inputs: Expected returns (μ), covariance matrix (Σ), risk aversion (λ)
- Output: Optimal weights (w)
- Problems: Sensitive to inputs, corner solutions, estimation error

## Black-Litterman
- Starts from market equilibrium (CAPM implied returns)
- Investor expresses views: "Asset A will outperform B by 3%"
- Blends equilibrium + views → posterior expected returns
- More stable and intuitive weights than pure MVO

## Constraints for Real-world Portfolios
1. **Long-only**: w_i >= 0
2. **Full investment**: Σw_i = 1.0
3. **Position limits**: w_i <= 0.05 (5%)
4. **Sector limits**: Σw_sector <= 0.30
5. **Turnover constraint**: Σ|w_new - w_old| <= 0.50
6. **Minimum position**: w_i >= 0.005 or 0

## Risk Budgeting
- Assign risk budget to each asset/strategy
- Risk contribution: w_i * (Σw)_i / σ_portfolio
- Equal risk contribution (ERC) as robust alternative to MVO

## A-Share Considerations
- CSI 300 for equity universe
- Include bond ETFs/convertible bonds for diversification
- Liquidity filter: exclude stocks with daily turnover < 10M CNY
- Rebalance monthly or quarterly to minimize transaction costs
