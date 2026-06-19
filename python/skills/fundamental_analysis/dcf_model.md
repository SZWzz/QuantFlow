---
title: DCF Valuation Model
category: fundamental_analysis
tags: [dcf, valuation, intrinsic-value, wacc]
difficulty: advanced
---

# DCF (Discounted Cash Flow) Model

## Framework
1. Project free cash flows (5-10 years)
2. Calculate terminal value
3. Discount to present value using WACC
4. Subtract net debt → equity value
5. Divide by shares outstanding → intrinsic value per share

## Free Cash Flow Formula
```
FCF = EBIT * (1 - tax_rate)
    + Depreciation & Amortization
    - Capex
    - Change in Working Capital
```

## Key Assumptions
- **Revenue growth**: 3-5 year analyst consensus, then fade to GDP growth
- **Margins**: Mean-revert to industry average by year 5
- **WACC**: Typically 8-12% for A-shares
- **Terminal growth**: 2-3% (no higher than long-term GDP growth)

## Terminal Value Methods
1. **Gordon Growth**: TV = FCF_last * (1+g) / (WACC - g)
2. **Exit Multiple**: TV = EBITDA_last * industry_multiple

## Margin of Safety
- Buy only when intrinsic value > market price * 1.3 (30% margin)
- Wider margin for: cyclical industries, high debt, uncertain growth
