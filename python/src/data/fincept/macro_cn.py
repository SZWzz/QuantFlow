"""
AKShare China Economics Data Wrapper
Wrapper for Chinese economic indicators and macro data
Returns JSON output for Qt/C++ integration

NOTE: Contains 85 endpoints total.

FAST endpoints (54): Response time < 20 seconds
gdp, gdp_yearly, cpi, ppi, pmi, non_man_pmi, money_supply, shibor_all, lpr, reserve_requirement_ratio,
new_financial_credit, bank_financing, central_bank_balance, fdi, fx_reserves_yearly, fx_gold, rmb,
construction_index, enterprise_boom_index, cx_pmi_yearly, cx_services_pmi_yearly, real_estate, new_house_price,
consumer_goods_retail, daily_energy, commodity_price_index, au_report, urban_unemployment, national_tax_receipts,
czsr, gdzctz, hgjck, gyzjz, qyspjg, wbck, whxd, xfzxx, stock_market_cap, market_margin_sh, market_margin_sz,
insurance_income, freight_index, lpi_index, mobile_number, hk_cpi, hk_cpi_ratio, hk_ppi, hk_gbp, hk_gbp_ratio,
hk_rate_of_unemployment, hk_trade_diff_ratio, hk_building_amount, hk_building_volume, hk_market_info

SLOW endpoints (25): Response time 60-120 seconds - These work but take 1-2 minutes
cpi_monthly, cpi_yearly, ppi_yearly, pmi_yearly, m2_yearly, trade_balance, exports_yoy, imports_yoy,
industrial_production_yoy, construction_price_index, agricultural_index, agricultural_product, vegetable_basket,
energy_index, bond_public, bdti_index, bsi_index, yw_electronic_index

UNRELIABLE endpoints (4): Connection issues or server unavailable
supply_of_money, foreign_exchange_gold, retail_price_index, society_electricity, insurance, society_traffic_volume,
passenger_load_factor, postal_telecommunicational, international_tourism_fx, shrzgm

REQUIRES PARAMETERS (2): Cannot be auto-called without specific arguments
nbs_nation (needs: kind, path), nbs_region (needs: kind, path, region)

DEPRECATED (1): API structure changed in AkShare library
swap_rate
"""

import sys
import json
import re
import os

# Silence akshare's internal tqdm progress bars: some macro_china_* helpers
# emit progress to STDOUT, which corrupts the JSON CLI output that the Go
# sidecar parses via subprocess.run(). TQDM_DISABLE is tqdm's official opt-out.
os.environ.setdefault("TQDM_DISABLE", "1")

import pandas as pd
import akshare as ak
from typing import Dict, Any, List
from datetime import datetime, timedelta, date
import time


class DateTimeEncoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, (datetime, date)):
            return obj.isoformat()
        return super().default(obj)


class AKShareError:
    """Custom error class for AKShare API errors"""
    def __init__(self, endpoint: str, error: str, data_source: str = None):
        self.endpoint = endpoint
        self.error = error
        self.data_source = data_source
        self.timestamp = int(datetime.now().timestamp())

    def to_dict(self) -> Dict[str, Any]:
        return {
            "endpoint": self.endpoint,
            "error": self.error,
            "data_source": self.data_source,
            "timestamp": self.timestamp,
            "type": "AKShareError"
        }


class ChinaEconomicsWrapper:
    """China economics wrapper with VALIDATED akshare functions (85 total)"""

    # Field mapping for standardized extraction. Each akshare macro function
    # returns a DataFrame with Chinese business column names that differ per
    # endpoint — there is no universal "value" column. This table maps each
    # endpoint to (date_col, value_col, unit, display_name, category, polarity)
    # so that get_normalized() can produce a uniform {date, value, signal} series.
    #
    # Sort order also varies: gdp/cpi/ppi/pmi/money_supply return newest-first
    # (descending), while gdp_yearly/non_man_pmi/shibor_all/lpr return
    # oldest-first (ascending). get_normalized() sorts by a smart date key so
    # callers never have to guess the order.
    #
    # `polarity` drives the bullish/bearish signal semantics (standard finance
    # logic, consistent with the FRED signal rules in govdata_service.go):
    #   "positive" — value↑ → bullish  (GDP/PMI/M2 growth, trade surplus)
    #   "negative" — value↑ → bearish  (CPI/PPI inflation, unemployment, Shibor)
    #   "inverse"  — value↓ → bullish  (LPR rate cut = easing = bullish)
    #
    # Columns below were verified against akshare 1.18.64 on 2026-06-27 by
    # inspecting the actual DataFrame records. Unverified endpoints are
    # intentionally omitted; get_normalized() returns success=False for them
    # rather than silently extracting the wrong field.
    MACRO_CN_FIELDS: Dict[str, Dict[str, str]] = {
        # --- Core Indicators ---
        "gdp": {"date_col": "季度", "value_col": "国内生产总值-同比增长", "unit": "%", "name_cn": "GDP 同比增长率", "category": "Core Indicators", "polarity": "positive"},
        "gdp_yearly": {"date_col": "日期", "value_col": "今值", "unit": "%", "name_cn": "GDP 年率", "category": "Core Indicators", "polarity": "positive"},
        "cpi": {"date_col": "月份", "value_col": "全国-同比增长", "unit": "%", "name_cn": "CPI 同比", "category": "Core Indicators", "polarity": "negative"},
        "cpi_monthly": {"date_col": "日期", "value_col": "今值", "unit": "%", "name_cn": "CPI 月率", "category": "Core Indicators", "polarity": "negative"},
        "cpi_yearly": {"date_col": "日期", "value_col": "今值", "unit": "%", "name_cn": "CPI 年率", "category": "Core Indicators", "polarity": "negative"},
        "ppi": {"date_col": "月份", "value_col": "当月同比增长", "unit": "%", "name_cn": "PPI 同比", "category": "Core Indicators", "polarity": "negative"},
        "ppi_yearly": {"date_col": "日期", "value_col": "今值", "unit": "%", "name_cn": "PPI 年率", "category": "Core Indicators", "polarity": "negative"},
        "pmi": {"date_col": "月份", "value_col": "制造业-指数", "unit": "", "name_cn": "制造业 PMI", "category": "Core Indicators", "polarity": "positive"},
        "pmi_yearly": {"date_col": "日期", "value_col": "今值", "unit": "", "name_cn": "制造业 PMI 年率", "category": "Core Indicators", "polarity": "positive"},
        "non_man_pmi": {"date_col": "日期", "value_col": "今值", "unit": "", "name_cn": "非制造业 PMI", "category": "Core Indicators", "polarity": "positive"},
        # --- Monetary & Financial ---
        "money_supply": {"date_col": "月份", "value_col": "货币和准货币(M2)-同比增长", "unit": "%", "name_cn": "M2 同比增长", "category": "Monetary & Financial", "polarity": "positive"},
        "m2_yearly": {"date_col": "日期", "value_col": "今值", "unit": "%", "name_cn": "M2 年率", "category": "Monetary & Financial", "polarity": "positive"},
        "shibor_all": {"date_col": "日期", "value_col": "1Y-定价", "unit": "%", "name_cn": "Shibor 1年期", "category": "Monetary & Financial", "polarity": "negative"},
        "lpr": {"date_col": "TRADE_DATE", "value_col": "LPR1Y", "unit": "%", "name_cn": "LPR 1年期", "category": "Monetary & Financial", "polarity": "inverse"},
        # --- Trade & FX (英为财情 今值/前值 格式) ---
        "trade_balance": {"date_col": "日期", "value_col": "今值", "unit": "亿美元", "name_cn": "贸易差额", "category": "Trade & FX", "polarity": "positive"},
        "exports_yoy": {"date_col": "日期", "value_col": "今值", "unit": "%", "name_cn": "出口同比", "category": "Trade & FX", "polarity": "positive"},
        "imports_yoy": {"date_col": "日期", "value_col": "今值", "unit": "%", "name_cn": "进口同比", "category": "Trade & FX", "polarity": "positive"},
        # --- Industry & Production ---
        "enterprise_boom_index": {"date_col": "季度", "value_col": "企业景气指数-指数", "unit": "", "name_cn": "企业景气指数", "category": "Industry & Production", "polarity": "positive"},
        # --- Employment ---
        # urban_unemployment removed: akshare macro_china_urban_unemployment
        # returns non-JSON / broken response as of akshare 1.18.64 (2026-06-27).
        # Re-add after upstream is fixed.
    }

    # Endpoints used by the summary view. All endpoints with a MACRO_CN_FIELDS
    # mapping are included here — they were individually benchmarked on
    # 2026-06-27 and all complete in 1-3s, so a 5-worker ThreadPoolExecutor
    # finishes the full set in ~5s. Previously only 9 "FAST_CORE" endpoints
    # were fetched, which left 29 cards showing "no data" in the UI.
    FAST_CORE_ENDPOINTS = [
        "gdp", "gdp_yearly", "cpi", "cpi_monthly", "cpi_yearly",
        "ppi", "ppi_yearly", "pmi", "pmi_yearly", "non_man_pmi",
        "money_supply", "m2_yearly", "shibor_all", "lpr",
        "trade_balance", "exports_yoy", "imports_yoy",
        "enterprise_boom_index",
    ]

    def __init__(self):
        self.default_timeout = 30
        self.retry_delay = 2

    def _convert_dataframe_to_json_safe(self, df: pd.DataFrame) -> List[Dict[str, Any]]:
        """Convert DataFrame to JSON-safe format"""
        df_copy = df.copy()
        for col in df_copy.columns:
            if pd.api.types.is_datetime64_any_dtype(df_copy[col]):
                df_copy[col] = df_copy[col].astype(str)
        return df_copy.to_dict('records')

    def _safe_call_with_retry(self, func, *args, max_retries: int = 3, **kwargs) -> Dict[str, Any]:
        """Safely call AKShare function with retry logic"""
        last_error = None

        for attempt in range(max_retries):
            try:
                result = func(*args, **kwargs)

                if result is not None and hasattr(result, 'empty') and not result.empty:
                    return {
                        "success": True,
                        "data": self._convert_dataframe_to_json_safe(result),
                        "count": len(result),
                        "timestamp": int(datetime.now().timestamp()),
                        "source": f"akshare.{func.__name__}"
                    }

                return {
                    "success": False,
                    "error": "No data returned",
                    "data": [],
                    "count": 0,
                    "timestamp": int(datetime.now().timestamp())
                }

            except ValueError as e:
                error_msg = str(e)
                if "Length mismatch" in error_msg or "Expected axis" in error_msg:
                    return {"success": False, "error": "AKShare API structure changed. Endpoint unavailable.", "data": [], "error_type": "api_mismatch", "timestamp": int(datetime.now().timestamp())}
                last_error = error_msg
                if attempt < max_retries - 1:
                    time.sleep(self.retry_delay)
                continue
            except KeyError as e:
                return {"success": False, "error": f"Missing field: {str(e)}", "data": [], "error_type": "missing_field", "timestamp": int(datetime.now().timestamp())}
            except (ConnectionError, TimeoutError) as e:
                last_error = str(e)
                if attempt < max_retries - 1:
                    time.sleep(self.retry_delay * 2)
                continue
            except Exception as e:
                last_error = str(e)
                if attempt < max_retries - 1:
                    time.sleep(self.retry_delay)
                continue

        error_obj = AKShareError(
            endpoint=func.__name__,
            error=last_error or "Unknown error",
            data_source=getattr(func, '__module__', 'unknown')
        )
        return {
            "success": False,
            "error": error_obj.to_dict(),
            "data": [],
            "count": 0,
            "timestamp": int(datetime.now().timestamp())
        }

    # ==================== CORE INDICATORS ====================

    def get_gdp(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_gdp)

    def get_gdp_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_gdp_yearly)

    def get_cpi(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_cpi)

    def get_cpi_monthly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_cpi_monthly)

    def get_cpi_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_cpi_yearly)

    def get_ppi(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_ppi)

    def get_ppi_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_ppi_yearly)

    def get_pmi(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_pmi)

    def get_pmi_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_pmi_yearly)

    def get_non_man_pmi(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_non_man_pmi)

    # ==================== MONETARY & FINANCIAL ====================

    def get_money_supply(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_money_supply)

    def get_supply_of_money(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_supply_of_money)

    def get_m2_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_m2_yearly)

    def get_shibor_all(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_shibor_all)

    def get_lpr(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_lpr)

    def get_swap_rate(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_swap_rate)

    def get_reserve_requirement_ratio(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_reserve_requirement_ratio)

    def get_new_financial_credit(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_new_financial_credit)

    def get_bank_financing(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_bank_financing)

    def get_central_bank_balance(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_central_bank_balance)

    # ==================== TRADE & FX ====================

    def get_trade_balance(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_trade_balance)

    def get_exports_yoy(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_exports_yoy)

    def get_imports_yoy(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_imports_yoy)

    def get_fdi(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_fdi)

    def get_fx_reserves_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_fx_reserves_yearly)

    def get_fx_gold(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_fx_gold)

    def get_foreign_exchange_gold(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_foreign_exchange_gold)

    def get_rmb(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_rmb)

    # ==================== INDUSTRY & PRODUCTION ====================

    def get_industrial_production_yoy(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_industrial_production_yoy)

    def get_construction_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_construction_index)

    def get_construction_price_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_construction_price_index)

    def get_enterprise_boom_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_enterprise_boom_index)

    def get_cx_pmi_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_cx_pmi_yearly)

    def get_cx_services_pmi_yearly(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_cx_services_pmi_yearly)

    # ==================== REAL ESTATE ====================

    def get_real_estate(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_real_estate)

    def get_new_house_price(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_new_house_price)

    # ==================== CONSUMPTION & RETAIL ====================

    def get_consumer_goods_retail(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_consumer_goods_retail)

    def get_retail_price_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_retail_price_index)

    # ==================== AGRICULTURE ====================

    def get_agricultural_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_agricultural_index)

    def get_agricultural_product(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_agricultural_product)

    def get_vegetable_basket(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_vegetable_basket)

    # ==================== ENERGY & COMMODITIES ====================

    def get_daily_energy(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_daily_energy)

    def get_energy_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_energy_index)

    def get_society_electricity(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_society_electricity)

    def get_commodity_price_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_commodity_price_index)

    def get_au_report(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_au_report)

    # ==================== EMPLOYMENT ====================

    def get_urban_unemployment(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_urban_unemployment)

    # ==================== FISCAL ====================

    def get_national_tax_receipts(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_national_tax_receipts)

    def get_czsr(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_czsr)

    def get_gdzctz(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_gdzctz)

    def get_hgjck(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hgjck)

    def get_gyzjz(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_gyzjz)

    def get_qyspjg(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_qyspjg)

    def get_shrzgm(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_shrzgm)

    def get_wbck(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_wbck)

    def get_whxd(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_whxd)

    def get_xfzxx(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_xfzxx)

    # ==================== MARKETS ====================

    def get_stock_market_cap(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_stock_market_cap)

    def get_market_margin_sh(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_market_margin_sh)

    def get_market_margin_sz(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_market_margin_sz)

    def get_bond_public(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_bond_public)

    # ==================== INSURANCE & FINANCE ====================

    def get_insurance(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_insurance)

    def get_insurance_income(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_insurance_income)

    # ==================== TRANSPORTATION & LOGISTICS ====================

    def get_freight_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_freight_index)

    def get_bdti_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_bdti_index)

    def get_bsi_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_bsi_index)

    def get_lpi_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_lpi_index)

    def get_society_traffic_volume(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_society_traffic_volume)

    def get_passenger_load_factor(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_passenger_load_factor)

    # ==================== TELECOM & SERVICES ====================

    def get_postal_telecommunicational(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_postal_telecommunicational)

    def get_mobile_number(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_mobile_number)

    def get_international_tourism_fx(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_international_tourism_fx)

    def get_yw_electronic_index(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_yw_electronic_index)

    # ==================== NBS DATA ====================

    def get_nbs_nation(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_nbs_nation)

    def get_nbs_region(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_nbs_region)

    # ==================== HONG KONG ====================

    def get_hk_cpi(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_cpi)

    def get_hk_cpi_ratio(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_cpi_ratio)

    def get_hk_ppi(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_ppi)

    def get_hk_gbp(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_gbp)

    def get_hk_gbp_ratio(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_gbp_ratio)

    def get_hk_rate_of_unemployment(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_rate_of_unemployment)

    def get_hk_trade_diff_ratio(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_trade_diff_ratio)

    def get_hk_building_amount(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_building_amount)

    def get_hk_building_volume(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_building_volume)

    def get_hk_market_info(self) -> Dict[str, Any]:
        return self._safe_call_with_retry(ak.macro_china_hk_market_info)

    # ==================== UTILITY ====================

    def get_all_available_endpoints(self) -> Dict[str, Any]:
        """Get list of all available endpoints"""
        endpoints = [
            "gdp", "gdp_yearly", "cpi", "cpi_monthly", "cpi_yearly", "ppi", "ppi_yearly",
            "pmi", "pmi_yearly", "non_man_pmi", "money_supply", "supply_of_money", "m2_yearly",
            "shibor_all", "lpr", "swap_rate", "reserve_requirement_ratio", "new_financial_credit",
            "bank_financing", "central_bank_balance", "trade_balance", "exports_yoy", "imports_yoy",
            "fdi", "fx_reserves_yearly", "fx_gold", "foreign_exchange_gold", "rmb",
            "industrial_production_yoy", "construction_index", "construction_price_index",
            "enterprise_boom_index", "cx_pmi_yearly", "cx_services_pmi_yearly",
            "real_estate", "new_house_price", "consumer_goods_retail", "retail_price_index",
            "agricultural_index", "agricultural_product", "vegetable_basket",
            "daily_energy", "energy_index", "society_electricity", "commodity_price_index", "au_report",
            "urban_unemployment", "national_tax_receipts", "czsr", "gdzctz", "hgjck",
            "gyzjz", "qyspjg", "shrzgm", "wbck", "whxd", "xfzxx",
            "stock_market_cap", "market_margin_sh", "market_margin_sz", "bond_public",
            "insurance", "insurance_income", "freight_index", "bdti_index", "bsi_index",
            "lpi_index", "society_traffic_volume", "passenger_load_factor",
            "postal_telecommunicational", "mobile_number", "international_tourism_fx", "yw_electronic_index",
            "nbs_nation", "nbs_region",
            "hk_cpi", "hk_cpi_ratio", "hk_ppi", "hk_gbp", "hk_gbp_ratio",
            "hk_rate_of_unemployment", "hk_trade_diff_ratio", "hk_building_amount",
            "hk_building_volume", "hk_market_info"
        ]

        return {
            "success": True,
            "data": {
                "available_endpoints": endpoints,
                "total_count": len(endpoints),
                "categories": {
                    "Core Indicators": ["gdp", "cpi", "ppi", "pmi", "gdp_yearly", "cpi_yearly", "ppi_yearly"],
                    "Monetary & Financial": ["money_supply", "m2_yearly", "shibor_all", "lpr", "swap_rate"],
                    "Trade & FX": ["trade_balance", "exports_yoy", "imports_yoy", "fdi", "fx_reserves_yearly"],
                    "Industry & Production": ["industrial_production_yoy", "construction_index", "enterprise_boom_index"],
                    "Real Estate": ["real_estate", "new_house_price"],
                    "Consumption": ["consumer_goods_retail", "retail_price_index"],
                    "Agriculture": ["agricultural_index", "agricultural_product", "vegetable_basket"],
                    "Energy": ["daily_energy", "energy_index", "society_electricity"],
                    "Employment": ["urban_unemployment"],
                    "Markets": ["stock_market_cap", "market_margin_sh", "market_margin_sz"],
                    "Hong Kong": ["hk_cpi", "hk_gdp", "hk_ppi", "hk_market_info"],
                },
                "timestamp": int(datetime.now().timestamp())
            },
            "count": len(endpoints),
            "timestamp": int(datetime.now().timestamp())
        }

    # ==================== STANDARDIZED EXTRACTION ====================

    @staticmethod
    def _date_sort_key(s: str):
        """Build a comparable sort key from Chinese/ISO date strings.

        Handles: '2025-07-15' (ISO), '2008年01月份' (中文月份),
        '2006年第1季度' (中文季度), bare years, and falls back to the
        raw string so sorting never raises on an unexpected format.
        """
        s = str(s)
        # ISO date: 2025-07-15 or 2025-7
        m = re.match(r"^(\d{4})-(\d{1,2})(?:-(\d{1,2}))?", s)
        if m:
            return (int(m.group(1)), int(m.group(2)), int(m.group(3) or 1))
        # 中文月份: 2008年01月份 / 2008年1月
        m = re.match(r"^(\d{4})年(\d{1,2})月", s)
        if m:
            return (int(m.group(1)), int(m.group(2)), 1)
        # 中文季度: 2006年第1季度
        m = re.match(r"^(\d{4})年第(\d)季度", s)
        if m:
            q = int(m.group(2))
            return (int(m.group(1)), q * 3, 1)
        # 仅年份
        m = re.match(r"^(\d{4})", s)
        if m:
            return (int(m.group(1)), 0, 0)
        return (0, 0, 0)

    def get_normalized(self, endpoint: str, limit: int = 24) -> Dict[str, Any]:
        """Fetch an endpoint and return a uniform {date, value} series.

        akshare's macro_china_* functions each return DataFrames with
        different Chinese column names and inconsistent sort orders. This
        method uses :pyattr:`MACRO_CN_FIELDS` to extract the right columns,
        coerces values to float (skipping NaN/non-numeric rows), sorts the
        series ascending by date, and returns the latest point plus the
        trailing ``limit`` points for charting.

        Returns a dict with keys:
            success, endpoint, name_cn, unit, category,
            latest_value, latest_date, series, count, timestamp, source.
        On failure returns success=False with an explanatory ``error``.
        """
        fields = self.MACRO_CN_FIELDS.get(endpoint)
        if fields is None:
            return {
                "success": False,
                "error": f"no field mapping for endpoint '{endpoint}'",
                "endpoint": endpoint,
                "timestamp": int(datetime.now().timestamp()),
            }

        # Resolve the underlying getter (e.g. endpoint 'gdp' -> self.get_gdp)
        method_name = f"get_{endpoint}" if not endpoint.startswith("get_") else endpoint
        method = getattr(self, method_name, None)
        if method is None or not callable(method):
            return {
                "success": False,
                "error": f"unknown endpoint '{endpoint}' (method '{method_name}' not found)",
                "endpoint": endpoint,
                "timestamp": int(datetime.now().timestamp()),
            }

        raw = method()
        if not raw.get("success"):
            err = raw.get("error")
            # _safe_call_with_retry may nest the error in an AKShareError dict
            if isinstance(err, dict):
                err = err.get("error", str(err))
            return {
                "success": False,
                "error": f"upstream {method_name} failed: {err}",
                "endpoint": endpoint,
                "timestamp": int(datetime.now().timestamp()),
            }

        records = raw.get("data", [])
        if not records:
            return {
                "success": False,
                "error": "upstream returned no records",
                "endpoint": endpoint,
                "timestamp": int(datetime.now().timestamp()),
            }

        date_col = fields["date_col"]
        value_col = fields["value_col"]

        series: List[Dict[str, Any]] = []
        for r in records:
            if not isinstance(r, dict):
                continue
            d = r.get(date_col)
            v = r.get(value_col)
            if d is None or v is None:
                continue
            try:
                fv = float(v)
            except (TypeError, ValueError):
                continue
            # Skip NaN / inf produced by akshare's empty forecast cells
            try:
                if pd.isna(fv) or not pd.notna(fv):
                    continue
            except (TypeError, ValueError):
                pass
            series.append({"date": str(d), "value": fv})

        if not series:
            return {
                "success": False,
                "error": f"no valid rows extracted (date_col={date_col!r}, value_col={value_col!r})",
                "endpoint": endpoint,
                "timestamp": int(datetime.now().timestamp()),
            }

        # Sort ascending by date so series[-1] is always the newest point,
        # regardless of whether akshare returned newest-first or oldest-first.
        try:
            series.sort(key=lambda p: self._date_sort_key(p["date"]))
        except Exception:
            # If keys are incomparable, fall back to stable original order;
            # we then take the first record's reported "newest" heuristically.
            pass

        latest = series[-1]
        if limit and limit > 0 and len(series) > limit:
            series = series[-limit:]

        # ── Signal computation (standard finance logic) ──
        # direction/change from the last two points; signal from polarity.
        # polarity → bullish/bearish mapping:
        #   positive: value↑ → bullish (GDP/PMI/M2 growth)
        #   negative: value↑ → bearish (CPI/PPI/unemployment/Shibor)
        #   inverse:  value↓ → bullish (LPR rate cut = easing)
        latest_v = latest["value"]
        prev_v = series[-2]["value"] if len(series) >= 2 else latest_v
        change = 0.0
        if prev_v != 0:
            change = ((latest_v - prev_v) / abs(prev_v)) * 100
        change = round(change, 2)

        if change > 0.5:
            direction = "up"
        elif change < -0.5:
            direction = "down"
        else:
            direction = "flat"

        polarity = fields.get("polarity", "positive")
        if direction == "flat":
            signal = "neutral"
        elif polarity == "positive":
            signal = "bullish" if direction == "up" else "bearish"
        elif polarity == "negative":
            signal = "bearish" if direction == "up" else "bullish"
        else:  # inverse — rate cut (down) is bullish
            signal = "bullish" if direction == "down" else "bearish"

        return {
            "success": True,
            "endpoint": endpoint,
            "name_cn": fields.get("name_cn", endpoint),
            "unit": fields.get("unit", ""),
            "category": fields.get("category", ""),
            "polarity": polarity,
            "latest_value": latest_v,
            "latest_date": latest["date"],
            "change": change,
            "direction": direction,
            "signal": signal,
            "series": series,
            "count": len(series),
            "timestamp": int(datetime.now().timestamp()),
            "source": f"akshare.{method_name}",
        }

    def get_summary(self, series_limit: int = 12) -> Dict[str, Any]:
        """Fetch latest values + short series for all FAST core endpoints.

        Runs :pyattr:`FAST_CORE_ENDPOINTS` through :meth:`get_normalized` in
        parallel (ThreadPoolExecutor, 5 workers, 120s budget) and returns a
        single payload suitable for the macro panel summary view:

            {
              success, data: {
                available_endpoints, total_count, categories,
                values: {endpoint: {latest_value, latest_date, unit,
                                    name_cn, category, series}},
                timestamp
              }, count, timestamp
            }

        Endpoints that fail (network, akshare error, no mapping) are silently
        omitted from ``values`` so one bad source never blanks the whole panel.
        """
        import concurrent.futures

        info = self.get_all_available_endpoints()
        info_data = info.get("data", {}) if info.get("success") else {}
        categories = info_data.get("categories", {})
        available = info_data.get("available_endpoints", [])

        def _fetch(ep):
            return ep, self.get_normalized(ep, series_limit)

        values: Dict[str, Any] = {}
        with concurrent.futures.ThreadPoolExecutor(max_workers=5) as pool:
            futures = {pool.submit(_fetch, ep): ep for ep in self.FAST_CORE_ENDPOINTS}
            try:
                for f in concurrent.futures.as_completed(futures, timeout=120):
                    try:
                        ep, norm = f.result()
                    except Exception as exc:
                        continue
                    if norm.get("success"):
                        values[ep] = {
                            "latest_value": norm["latest_value"],
                            "latest_date": norm["latest_date"],
                            "unit": norm["unit"],
                            "name_cn": norm["name_cn"],
                            "category": norm["category"],
                            "change": norm["change"],
                            "direction": norm["direction"],
                            "signal": norm["signal"],
                            "polarity": norm["polarity"],
                            "series": norm["series"],
                        }
            except concurrent.futures.TimeoutError:
                # Keep whatever we collected before the deadline — partial data
                # is more useful to the user than an empty panel.
                pass

        return {
            "success": True,
            "data": {
                "available_endpoints": available,
                "total_count": len(available),
                "categories": categories,
                "values": values,
                "timestamp": int(datetime.now().timestamp()),
            },
            "count": len(values),
            "timestamp": int(datetime.now().timestamp()),
        }


# ==================== CLI ====================
if __name__ == "__main__":
    import sys
    import json

    # Get wrapper instance
    wrapper = ChinaEconomicsWrapper()

    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: python akshare_economics_china.py <endpoint> [args...]"}))
        sys.exit(1)

    endpoint = sys.argv[1]
    args = sys.argv[2:] if len(sys.argv) > 2 else []

    # Handle get_normalized <endpoint> [limit] — standardized {date,value} series
    if endpoint == "get_normalized":
        if len(args) < 1:
            print(json.dumps({"success": False, "error": "Usage: get_normalized <endpoint> [limit]"}))
            sys.exit(0)
        target = args[0]
        limit = 24
        if len(args) > 1:
            try:
                limit = int(args[1])
            except ValueError:
                pass
        result = wrapper.get_normalized(target, limit)
        print(json.dumps(result, ensure_ascii=True, cls=DateTimeEncoder))
        sys.exit(0)

    # Handle get_summary [limit] — one-shot parallel summary of fast core endpoints
    if endpoint == "get_summary":
        limit = 12
        if args:
            try:
                limit = int(args[0])
            except ValueError:
                pass
        result = wrapper.get_summary(limit)
        print(json.dumps(result, ensure_ascii=True, cls=DateTimeEncoder))
        sys.exit(0)

    # Handle get_all_endpoints
    if endpoint == "get_all_endpoints":
        if hasattr(wrapper, 'get_all_available_endpoints'):
            result = wrapper.get_all_available_endpoints()
        elif hasattr(wrapper, 'get_all_endpoints'):
            result = wrapper.get_all_endpoints()
        else:
            result = {"success": False, "error": "Endpoint list not available"}
        print(json.dumps(result, ensure_ascii=True))
        sys.exit(0)

    # Dynamic method resolution
    method_name = f"get_{endpoint}" if not endpoint.startswith("get_") else endpoint

    if hasattr(wrapper, method_name):
        method = getattr(wrapper, method_name)
        try:
            result = method(*args) if args else method()
            print(json.dumps(result, ensure_ascii=True, cls=DateTimeEncoder))
        except Exception as e:
            print(json.dumps({"success": False, "error": str(e), "endpoint": endpoint}))
    else:
        print(json.dumps({"success": False, "error": f"Unknown endpoint: {endpoint}. Method '{method_name}' not found."}))
