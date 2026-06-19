// Package market provides a unified symbol normalization layer for all data adapters.
// SymbolIdentity is the single source of truth for CN A-share stock identity,
// accepting any common format and providing adapter-specific conversions.
package market

import (
	"fmt"
	"strings"
)

// SymbolIdentity holds the canonical identifier for a CN A-share stock.
// The Code field is always a 6-digit string; Market is "SH", "SZ", or "BJ".
type SymbolIdentity struct {
	Raw    string // original input (for error messages)
	Code   string // 6-digit code, e.g. "600519"
	Market string // "SH" (Shanghai), "SZ" (Shenzhen), "BJ" (Beijing)
}

// NormalizeCN accepts any common CN stock identifier format and returns the
// canonical 6-digit code + market.
//
// Supported input formats:
//   - "600519"                          plain 6-digit code (market inferred from prefix)
//   - "600519.SH", "600519.SS"          with suffix
//   - "000001.SZ", "830799.BJ"          with suffix
//   - "sh600519", "SH600519"            with prefix
//   - "sz000001", "bj830799"            with prefix
func NormalizeCN(input string) (*SymbolIdentity, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return nil, fmt.Errorf("symbol: empty input")
	}

	id := &SymbolIdentity{Raw: input}

	// 1. Strip and detect suffix: .SH .SS .SZ .BJ
	upper := strings.ToUpper(s)
	if strings.HasSuffix(upper, ".SH") || strings.HasSuffix(upper, ".SS") {
		id.Code = strings.TrimSuffix(strings.TrimSuffix(upper, ".SH"), ".SS")
		id.Market = "SH"
	} else if strings.HasSuffix(upper, ".SZ") {
		id.Code = strings.TrimSuffix(upper, ".SZ")
		id.Market = "SZ"
	} else if strings.HasSuffix(upper, ".BJ") {
		id.Code = strings.TrimSuffix(upper, ".BJ")
		id.Market = "BJ"
	} else if strings.HasPrefix(upper, "SH") && len(upper) == 8 {
		id.Code = upper[2:]
		id.Market = "SH"
	} else if strings.HasPrefix(upper, "SZ") && len(upper) == 8 {
		id.Code = upper[2:]
		id.Market = "SZ"
	} else if strings.HasPrefix(upper, "BJ") && len(upper) == 8 {
		id.Code = upper[2:]
		id.Market = "BJ"
	} else if len(upper) == 6 && isAllDigits(upper) {
		id.Code = upper
		id.Market = marketFromCode(upper)
	} else {
		return nil, fmt.Errorf("symbol: unrecognized format %q", input)
	}

	// Validate code is 6 digits
	if len(id.Code) != 6 || !isAllDigits(id.Code) {
		return nil, fmt.Errorf("symbol: code %q is not 6 digits", id.Code)
	}

	// If market wasn't set by suffix/prefix, infer from code prefix
	if id.Market == "" {
		id.Market = marketFromCode(id.Code)
	}

	return id, nil
}

// ToEastMoney returns the EastMoney secid format: "1.600519" (SH) or "0.000001" (SZ/BJ).
func (s *SymbolIdentity) ToEastMoney() string {
	if s.Market == "SH" {
		return "1." + s.Code
	}
	return "0." + s.Code
}

// ToTencent returns the Tencent format: "sh600519" or "sz000001".
func (s *SymbolIdentity) ToTencent() string {
	return strings.ToLower(s.Market) + s.Code
}

// ToSina returns the Sina format: "sh600519" or "sz000001" (same as Tencent).
func (s *SymbolIdentity) ToSina() string {
	return strings.ToLower(s.Market) + s.Code
}

// ToBaidu returns the plain 6-digit code.
func (s *SymbolIdentity) ToBaidu() string {
	return s.Code
}

// ToMootdx returns the plain 6-digit code (mootdx / Python side normalizes internally).
func (s *SymbolIdentity) ToMootdx() string {
	return s.Code
}

// ToYahoo returns the Yahoo Finance format: "600519.SS" (SH) or "000001.SZ" (SZ).
func (s *SymbolIdentity) ToYahoo() string {
	switch s.Market {
	case "SH":
		return s.Code + ".SS"
	case "SZ":
		return s.Code + ".SZ"
	case "BJ":
		return s.Code + ".BJ"
	default:
		return s.Code + "." + s.Market
	}
}

// ToPlain returns the raw 6-digit code.
func (s *SymbolIdentity) ToPlain() string {
	return s.Code
}

// MarketCode returns "1" for Shanghai, "0" for Shenzhen/Beijing (EastMoney convention).
func (s *SymbolIdentity) MarketCode() string {
	if s.Market == "SH" {
		return "1"
	}
	return "0"
}

// String returns a human-readable representation.
func (s *SymbolIdentity) String() string {
	return fmt.Sprintf("%s.%s", s.Code, s.Market)
}

// marketFromCode infers the market from the first digit of a 6-digit code.
func marketFromCode(code string) string {
	if len(code) < 1 {
		return ""
	}
	switch code[0] {
	case '5', '6', '9':
		return "SH"
	case '0', '3':
		return "SZ"
	case '8', '4':
		return "BJ"
	default:
		return ""
	}
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
