// Package normalize rewrites dynamic values inside log messages into stable
// placeholders so that semantically identical errors can be clustered.
//
// Rule order matters: more specific patterns (URLs, IPv6, IP:port) run before
// the generic numeric fallback. Patterns that carry diagnostic meaning (HTTP
// status codes) are protected from over-normalization.
package normalize

import (
	"regexp"
	"strings"
)

// Normalization placeholder tokens.
const (
	tokenURL    = "<URL>"
	tokenIP     = "<IP>"
	tokenIPPort = "<IP>:<PORT>"
	tokenUUID   = "<UUID>"
	tokenTime   = "<TIME>"
	tokenPath   = "<PATH>"
	tokenHex    = "<HEX>"
	tokenNumber = "<NUMBER>"
)

// Status-code protection: "HTTP 500" is normalized away from the numeric rule
// so status codes keep their diagnostic meaning. "HTTP 500" becomes
// "HTTP _STATUS_500_" during normalization (the underscores block \b\d+\b)
// and is restored afterwards. Note the ${2} braces: "$2_" would be parsed as
// a capture-group named "2_" and expand to nothing.
var (
	statusRe     = regexp.MustCompile(`\b(HTTP|status)\s*[=:]?\s*(\d{3})\b`)
	statusBackRe = regexp.MustCompile(`_STATUS_(\d{3})_`)
	versionRe    = regexp.MustCompile(`\bHTTP/\d(?:\.\d)?\b`)

	urlRe      = regexp.MustCompile(`\bhttps?://[^\s<]+`)
	ipv6Re     = regexp.MustCompile(`\b[0-9a-fA-F:]+\b`)
	clockRe    = regexp.MustCompile(`\b\d{1,2}:\d{2}(?::\d{2}(?:\.\d+)?)?\b`)
	ipPortRe   = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{2,5}\b`)
	ipRe       = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	uuidRe     = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	timeRe     = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:?\d{2})?\b`)
	unixPathRe = regexp.MustCompile(`(?:/[a-zA-Z0-9_.-]+){2,}`)
	winPathRe  = regexp.MustCompile(`[A-Za-z]:\\(?:[^\\]+\\)+[^\\\s]+`)
	hexRe      = regexp.MustCompile(`\b[0-9a-fA-F]{8,}\b`)
	numRe      = regexp.MustCompile(`\b\d+\b`)
)

// Normalizer rewrites dynamic values in messages into stable placeholders.
type Normalizer struct{}

// New returns a Normalizer with the default rule set.
func New() *Normalizer { return &Normalizer{} }

// Normalize replaces dynamic values in msg with placeholders. Messages that
// contain no dynamic values are returned unchanged.
//
// Rule order matters: version strings and status codes are normalized first,
// then URL / timestamp / address patterns (most specific), and the generic
// numeric rule runs last.
func (n *Normalizer) Normalize(msg string) string {
	// "HTTP/1.1" version strings carry no diagnostic meaning per version;
	// collapse them to "HTTP" so HTTP/1.0 and HTTP/1.1 5xx errors cluster
	// together and the status code stays protectable by statusRe.
	msg = versionRe.ReplaceAllString(msg, "HTTP")

	// Protect HTTP status codes from the numeric rule.
	msg = statusRe.ReplaceAllString(msg, "$1 _STATUS_${2}_")

	msg = urlRe.ReplaceAllString(msg, tokenURL)
	// Full timestamps first, then bare wall-clock times, so the clock rule
	// never tears apart "2026-08-31 14:32:01" before it is consumed whole.
	msg = timeRe.ReplaceAllString(msg, tokenTime)
	msg = clockRe.ReplaceAllString(msg, tokenTime)

	msg = replaceIPv6(msg)
	msg = ipPortRe.ReplaceAllString(msg, tokenIPPort)
	msg = ipRe.ReplaceAllString(msg, tokenIP)
	msg = uuidRe.ReplaceAllString(msg, tokenUUID)
	msg = unixPathRe.ReplaceAllString(msg, tokenPath)
	msg = winPathRe.ReplaceAllString(msg, tokenPath)
	msg = hexRe.ReplaceAllString(msg, tokenHex)
	msg = numRe.ReplaceAllString(msg, tokenNumber)

	// Restore protected HTTP status codes.
	msg = statusBackRe.ReplaceAllString(msg, "$1")
	return msg
}

// replaceIPv6 normalizes IPv6 addresses but only when the token actually
// looks like one (colon-separated hex groups).
func replaceIPv6(msg string) string {
	return ipv6Re.ReplaceAllStringFunc(msg, func(m string) string {
		if isIPv6Like(m) {
			return tokenIP
		}
		return m
	})
}

// isIPv6Like reports whether s is a plausible IPv6 address: at least two
// colon-separated groups of up to four hex digits (empty groups allowed for
// the "::" compression).
func isIPv6Like(s string) bool {
	if strings.Count(s, ":") < 2 {
		return false
	}
	for _, part := range strings.Split(s, ":") {
		if len(part) > 4 {
			return false
		}
		for _, r := range part {
			if !isHexDigit(r) {
				return false
			}
		}
	}
	return true
}

// isHexDigit reports whether r is a hexadecimal digit.
func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
