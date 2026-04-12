package events

import "strings"

// stripeMinorUnitMultiplier returns the multiplier needed to convert a major-unit
// float price (e.g. 10.50) into Stripe's integer amount in the currency's minor
// unit.
//
// Most currencies use 2 decimals (multiplier 100). Some are zero-decimal
// (multiplier 1). A small set use 3 decimals (multiplier 1000), such as TND.
func stripeMinorUnitMultiplier(currency string) int64 {
	code := strings.ToUpper(strings.TrimSpace(currency))
	switch code {
	// 3-decimal currencies
	case "BHD", "JOD", "KWD", "OMR", "TND":
		return 1000
	// zero-decimal currencies
	case "BIF", "CLP", "DJF", "GNF", "JPY", "KMF", "KRW", "MGA", "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF":
		return 1
	default:
		return 100
	}
}
