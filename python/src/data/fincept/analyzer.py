"""
Financial Deep Analysis Module
===============================
Pure computation — receives pre-fetched financial data from Go,
returns structured analysis results. No network I/O.

Usage (via fetcher.py subprocess):
  python -m src.data.fincept.analyzer report_analysis '<json>'
  python -m src.data.fincept.analyzer valuation '<json>' '<quote_json>'
  python -m src.data.fincept.analyzer audit '<json>'
  python -m src.data.fincept.analyzer forecast '<json>'
"""
import json
import math
import sys


def _f(val):
    try: return float(val)
    except: return float('nan')


def _load(data):
    if isinstance(data, (dict, list)): return data
    if isinstance(data, bytes): data = data.decode("utf-8")
    return json.loads(data) if isinstance(data, str) and data else {}


# ══════ Module 1: Report Analysis ══════

def analyze_report(financials_json):
    data = _load(financials_json)
    income = _load(data.get("income", "[]")) if isinstance(data, dict) else []
    balance = _load(data.get("balance", "[]")) if isinstance(data, dict) else []

    income_by_p, balance_by_p = {}, {}
    for row in (income if isinstance(income, list) else []):
        p = row.get("报告期", row.get("period", ""))
        if p: income_by_p[p] = row
    for row in (balance if isinstance(balance, list) else []):
        p = row.get("报告期", row.get("period", ""))
        if p: balance_by_p[p] = row

    all_p = sorted(set(list(income_by_p.keys()) + list(balance_by_p.keys())), reverse=True)
    periods, anomalies = [], []

    for p in all_p[:12]:
        inc = income_by_p.get(p, {})
        bal = balance_by_p.get(p, {})
        rev = _f(inc.get("营业总收入", inc.get("revenue", 0)))
        np_ = _f(inc.get("净利润", inc.get("net_income", 0)))
        pp = _f(inc.get("归母净利润", inc.get("net_income_parent", np_)))
        ta = _f(bal.get("总资产", bal.get("total_assets", 0)))
        tl = _f(bal.get("总负债", bal.get("total_liabilities", 0)))
        eq = _f(bal.get("股东权益", bal.get("equity", 0)))
        gw = _f(bal.get("商誉", bal.get("goodwill", 0)))
        ar = _f(bal.get("应收账款", bal.get("receivables", 0)))

        periods.append({
            "period": p, "revenue": round(rev, 2), "net_profit": round(np_, 2),
            "parent_profit": round(pp, 2), "total_assets": round(ta, 2), "equity": round(eq, 2),
            "roe": round(np_ / eq * 100, 1) if eq > 0 else None,
            "debt_ratio": round(tl / ta * 100, 1) if ta > 0 else None,
            "profit_margin": round(np_ / rev * 100, 1) if rev > 0 else None,
        })

        if eq > 0 and gw / eq > 0.3:
            anomalies.append({"period": p, "type": "商誉占比过高", "level": "high",
                "detail": f"商誉/净资产={gw/eq*100:.1f}%，超30%阈值"})
        if ta > 0 and tl / ta > 0.7:
            anomalies.append({"period": p, "type": "负债率偏高", "level": "medium",
                "detail": f"资产负债率={tl/ta*100:.1f}%，超70%"})
        if rev > 0 and ar / rev > 0.5:
            anomalies.append({"period": p, "type": "应收占比过高", "level": "high",
                "detail": f"应收账款/营收={ar/rev*100:.1f}%，回款压力大"})

    score = 50
    if periods:
        latest = periods[0]
        roe = latest.get("roe")
        dr = latest.get("debt_ratio")
        pm = latest.get("profit_margin")
        if roe and roe > 15: score += 15
        elif roe is not None and roe > 8: score += 8
        elif roe is not None and roe < 0: score -= 15
        if dr and dr < 40: score += 10
        elif dr and dr > 70: score -= 10
        if pm and pm > 20: score += 10
        elif pm is not None and pm < 5: score -= 10
        if len(periods) >= 4:
            revs = [p.get("revenue", 0) or 0 for p in periods[:4]]
            if revs[0] > revs[3] * 1.2: score += 10
            elif revs[0] < revs[3]: score -= 5
    score -= sum(3 for a in anomalies if a["level"] == "high") + sum(1 for a in anomalies if a["level"] == "medium")
    score = max(0, min(100, score))
    grade = "优秀" if score >= 80 else "良好" if score >= 60 else "一般" if score >= 40 else "较差"

    return {"periods": periods[:12], "health_score": score, "health_grade": grade,
            "anomaly_flags": anomalies[:20], "metrics": {
                "latest_roe": periods[0].get("roe") if periods else None,
                "latest_debt_ratio": periods[0].get("debt_ratio") if periods else None,
                "latest_profit_margin": periods[0].get("profit_margin") if periods else None,
            }}


# ══════ Module 2: Valuation ══════

def compute_valuation(financials_json, quote_json=None):
    data = _load(financials_json)
    quote = _load(quote_json) if quote_json else {}
    cf = _load(data.get("cashflow", "[]")) if isinstance(data, dict) else []
    balance = _load(data.get("balance", "[]")) if isinstance(data, dict) else []

    fcf, net_debt, shares = 0, 0, 1
    if cf and isinstance(cf, list) and cf:
        lc = cf[0]
        fcf = _f(lc.get("自由现金流", lc.get("free_cash_flow", 0)))
        if fcf <= 0 or math.isnan(fcf):
            op = _f(lc.get("经营现金流净额", lc.get("operating_cash_flow", 0)))
            cx = abs(_f(lc.get("资本支出", lc.get("capex", 0))))
            fcf = op - cx if op > 0 else (op * 0.7 if op > 0 else 0)
        if fcf <= 0 or math.isnan(fcf):
            return {"error": "自由现金流为负或不可用，无法进行 DCF 估值", "scenarios": {}, "buy_sell": {}}
    if balance and isinstance(balance, list) and balance:
        lb = balance[0]
        net_debt = _f(lb.get("总负债", lb.get("total_liabilities", 0))) - _f(lb.get("货币资金", lb.get("cash", 0)))
    if quote:
        shares = _f(quote.get("total_shares", quote.get("shares_outstanding", 0)))
        if shares <= 0:
            mcap = _f(quote.get("market_cap", 0))
            price = _f(quote.get("price", 0))
            shares = mcap / price if price > 0 else 1
    if fcf <= 0:
        return {"error": "自由现金流为负，无法 DCF 估值", "scenarios": {}, "buy_sell": {}}

    wacc, g0, years = 0.08, 0.03, 5
    scenarios = {}
    for name, gm in [("保守", 0.5), ("基准", 1.0), ("乐观", 1.5)]:
        g = g0 * gm; pv = 0; f = fcf
        for yr in range(1, years + 1):
            f *= (1 + g); pv += f / ((1 + wacc) ** yr)
        tv = f * (1 + g) / (wacc - g) if wacc > g else f / 0.02
        pv_terminal = tv / ((1 + wacc) ** years)
        ev = pv + pv_terminal - net_debt
        scenarios[name] = {"growth_rate": round(g*100, 1), "value_per_share": round(ev / shares, 2) if shares > 0 else 0}

    fair = scenarios["基准"]["value_per_share"]
    cp = _f(quote.get("price", 0))
    bs = {}
    if cp > 0 and fair > 0:
        up = (fair / cp - 1) * 100
        bs = {"current_price": cp, "fair_value": fair, "upside_pct": round(up, 1),
              "suggestion": "买入" if up > 20 else "增持" if up > 5 else "持有" if up > -10 else "减持"}

    return {"scenarios": scenarios, "fcf": round(fcf, 2), "wacc_pct": 8, "buy_sell": bs}


# ══════ Module 3: Audit ══════

def detect_audit_risks(financials_json):
    data = _load(financials_json)
    inc = _load(data.get("income", "[]")) if isinstance(data, dict) else []
    bal = _load(data.get("balance", "[]")) if isinstance(data, dict) else []
    cf = _load(data.get("cashflow", "[]")) if isinstance(data, dict) else []

    li = inc[0] if isinstance(inc, list) and inc else {}
    lb = bal[0] if isinstance(bal, list) and bal else {}
    lc = cf[0] if isinstance(cf, list) and cf else {}

    rev = _f(li.get("营业总收入", li.get("revenue", 0)))
    findings = []

    ar = _f(lb.get("应收账款", lb.get("receivables", 0)))
    if rev > 0:
        r = ar / rev
        findings.append({"metric": "应收/营收比", "level": "high" if r > 0.5 else "medium" if r > 0.3 else "low",
            "value": f"{r*100:.1f}%", "threshold": ">30% 中, >50% 高", "detail": "高应收占比可能意味着收入质量差或回款周期长"})

    gw = _f(lb.get("商誉", lb.get("goodwill", 0)))
    eq = _f(lb.get("股东权益", lb.get("equity", 0)))
    if eq > 0:
        r = gw / eq
        findings.append({"metric": "商誉/净资产比", "level": "high" if r > 0.3 else "medium" if r > 0.15 else "low",
            "value": f"{r*100:.1f}%", "threshold": ">15% 中, >30% 高", "detail": "商誉占比过高意味着并购溢价风险，可能面临减值"})

    np_ = _f(li.get("净利润", li.get("net_income", 0)))
    ocf = _f(lc.get("经营现金流净额", lc.get("operating_cash_flow", 0)))
    if np_ > 0:
        r = ocf / np_
        findings.append({"metric": "经营现金流/净利润", "level": "high" if r < 0.5 else "medium" if r < 0.8 else "low",
            "value": f"{r*100:.1f}%", "threshold": "<80% 中, <50% 高", "detail": "经营现金流远低于利润可能意味着利润质量差或应收虚增"})

    sd = _f(lb.get("短期借款", lb.get("short_term_debt", 0)))
    ld = _f(lb.get("长期借款", lb.get("long_term_debt", 0)))
    ta = _f(lb.get("总资产", lb.get("total_assets", 0)))
    if ta > 0 and (sd + ld) > 0:
        r = (sd + ld) / ta
        findings.append({"metric": "有息负债/总资产", "level": "high" if r > 0.5 else "medium" if r > 0.3 else "low",
            "value": f"{r*100:.1f}%", "threshold": ">30% 中, >50% 高", "detail": "有息负债过高意味着杠杆风险，利率上升面临偿付压力"})

    risk_score = sum(3 for f in findings if f["level"] == "high") + sum(1 for f in findings if f["level"] == "medium")
    return {"findings": findings, "risk_score": risk_score,
            "risk_grade": "高风险" if risk_score >= 8 else "中等风险" if risk_score >= 4 else "低风险",
            "high_count": sum(1 for f in findings if f["level"] == "high"),
            "medium_count": sum(1 for f in findings if f["level"] == "medium")}


# ══════ Module 4: Forecast ══════

def forecast_financials(financials_json):
    data = _load(financials_json)
    income = _load(data.get("income", "[]")) if isinstance(data, dict) else []
    if not isinstance(income, list) or not income: return {"error": "No income data", "forecast_table": []}

    revs, pros, labels = [], [], []
    for row in sorted(income, key=lambda r: r.get("报告期", r.get("period", ""))):
        p = row.get("报告期", row.get("period", ""))
        rv = _f(row.get("营业总收入", row.get("revenue", 0)))
        pr = _f(row.get("净利润", row.get("net_income", 0)))
        if rv > 0 and p: labels.append(p); revs.append(rv); pros.append(pr)

    if len(revs) < 3: return {"error": "Need >=3 periods", "forecast_table": []}

    grs = [(revs[i]/revs[i-1]-1) for i in range(1, len(revs)) if revs[i-1] > 0]
    avg = sum(grs)/len(grs) if grs else 0.05

    table = []
    for s, m in [("保守",0.5),("基准",1.0),("乐观",1.5)]:
        g = avg * m
        y1r = round(revs[-1]*(1+g), 2); y2r = round(y1r*(1+g), 2)
        y1p = round(pros[-1]*(1+g), 2) if pros[-1] else 0; y2p = round(y1p*(1+g), 2)
        table.append({"scenario": s, "growth": f"{g*100:.1f}%", "y1_rev": y1r, "y2_rev": y2r, "y1_profit": y1p, "y2_profit": y2p})

    return {"periods": len(labels), "segments": [
        {"name": "主营收入", "pct": 85, "value": round(revs[-1]*0.85, 2)},
        {"name": "其他收入", "pct": 15, "value": round(revs[-1]*0.15, 2)}],
        "forecast_table": table, "latest_rev": revs[-1], "latest_profit": pros[-1], "avg_growth": round(avg*100, 1)}


# ══════ CLI ══════

# Export names must match data_type in fetcher.py
report_analysis = analyze_report
valuation = compute_valuation
audit = detect_audit_risks
forecast = forecast_financials

_DISPATCH = {"report_analysis": report_analysis, "valuation": valuation,
             "audit": audit, "forecast": forecast}

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print(json.dumps({"error": "Usage: analyzer.py <func> <json>"}))
        sys.exit(1)
    func = _DISPATCH.get(sys.argv[1])
    if not func:
        print(json.dumps({"error": f"Unknown: {sys.argv[1]}"})); sys.exit(1)
    try:
        args = sys.argv[2:]
        result = func(*args) if len(args) > 1 else func(args[0])
        print(json.dumps(result, default=str, ensure_ascii=False))
    except Exception as e:
        print(json.dumps({"error": str(e)})); sys.exit(1)
