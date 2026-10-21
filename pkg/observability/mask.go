package observability

import "regexp"

// msisdnPattern matches Iranian mobile numbers in the common local (11
// digits starting 09), international (+98 or 0098 followed by 9 and 9
// digits) forms. It intentionally does not try to validate operator
// prefixes — ARCHITECTURE.md §10 only requires masking in logs, not
// validation (that's pkg/msisdn's job).
var msisdnPattern = regexp.MustCompile(`(?:\+98|0098|0)9\d{9}`)

// MaskMSISDN replaces every Iranian MSISDN found in s with its masked form
// per ARCHITECTURE.md §10 ("Logs mask phone numbers (0912***4567)"): the
// first 4 digits and last 4 digits of the national significant number are
// kept, with the middle collapsed to a fixed "***". Any leading country
// code / trunk prefix is normalized away before masking so the masked
// output always has the documented 0912***4567 shape.
func MaskMSISDN(s string) string {
	return msisdnPattern.ReplaceAllStringFunc(s, maskOneMSISDN)
}

// maskOneMSISDN masks a single matched MSISDN. msisdnPattern only matches
// a leading "0", "+98", or "0098" immediately followed by 9 + 9 digits —
// a bare "98" with no such prefix is never matched on its own. So the
// match is one of:
//
//	0912xxxxxxx     (11 digits, local)
//	+989120001234
//	00989120001234
//
// In every case the "national" part is the last 10 digits (9 + 9 digits),
// which is normalized to the local 0-prefixed 11-digit form before masking
// so output is always e.g. 0912***1234.
func maskOneMSISDN(match string) string {
	// Keep only the trailing 10 digits: "9" + the 9 subscriber digits.
	national := match
	if len(national) > 10 {
		national = national[len(national)-10:]
	}
	local := "0" + national // 0912xxxxxxx, 11 digits

	const prefixLen = 4 // "0912"
	const suffixLen = 4 // last 4 digits
	if len(local) < prefixLen+suffixLen {
		// Defensive: shouldn't happen given the regex, but never panic on
		// log input.
		return match
	}
	return local[:prefixLen] + "***" + local[len(local)-suffixLen:]
}
