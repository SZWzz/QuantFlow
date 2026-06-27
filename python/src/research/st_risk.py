"""ST/*ST delisting risk prediction for A-shares.

Based on exchange listing rules (SSE/SZSE):
- Main Board: revenue < 300M AND net profit < 0 → ST
- ChiNext/STAR: revenue < 100M AND net profit < 0 → ST
- Net assets < 0 → *ST
- Audit opinion: adverse/disclaimer → ST
"""

def predict_st_risk(symbol: str, revenue: float = None, net_profit: float = None,
                    net_assets: float = None, audit_opinion: str = None,
                    consecutive_losses: int = 0) -> dict:
    """Predict ST/*ST delisting risk.
    
    Returns:
        dict with keys: risk_level ('high'|'medium'|'low'), confidence (0-1), reasons (list[str])
    """
    reasons = []
    
    # Determine board thresholds
    if symbol.startswith(('300', '301', '688')):
        rev_threshold = 100_000_000  # 1 亿
    else:
        rev_threshold = 300_000_000  # 3 亿
    
    # R1: Revenue + Net Profit check
    if revenue is not None and net_profit is not None:
        if revenue < rev_threshold and net_profit < 0:
            reasons.append(f"R1: revenue {revenue/1e8:.2f}亿 < {rev_threshold/1e8:.1f}亿 threshold AND net profit negative")
    
    # R2: Net Assets check
    if net_assets is not None and net_assets < 0:
        reasons.append("R2: net assets negative (*ST risk)")
    
    # R3: Audit check  
    if audit_opinion in ('adverse', 'disclaimer', 'qualified'):
        reasons.append(f"E1: audit opinion '{audit_opinion}'")
    
    # R4: Consecutive losses
    if consecutive_losses >= 3:
        reasons.append(f"R4: {consecutive_losses} consecutive years of losses")
    elif consecutive_losses >= 2:
        reasons.append(f"R4: {consecutive_losses} consecutive years of losses (approaching)")
    
    if not reasons:
        return {'risk_level': 'low', 'confidence': 0.9, 'reasons': []}
    
    # Classify risk level
    has_star = any('*ST' in r for r in reasons)
    critical_count = len([r for r in reasons if r.startswith('R')])
    
    if has_star or critical_count >= 2:
        return {'risk_level': 'high', 'confidence': 0.8, 'reasons': reasons}
    elif critical_count >= 1 or len(reasons) >= 2:
        return {'risk_level': 'medium', 'confidence': 0.6, 'reasons': reasons}
    else:
        return {'risk_level': 'low', 'confidence': 0.7, 'reasons': reasons}
