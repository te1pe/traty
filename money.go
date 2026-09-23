package main

import (
	"errors"
	"strconv"
	"strings"
)

// Money is stored as an integer number of minor units (cents, kopecks).
// All supported currencies have exactly 2 decimal places.
const minorUnits = 100

// MaxAmountMinor is 9 999 999,99 in minor units.
const MaxAmountMinor int64 = 999999999

// nbsp is the non-breaking space used as a thousands separator and before
// the currency sign, so an amount never wraps across lines.
const nbsp = " "

// Currency describes one supported currency.
type Currency struct {
	Code   string
	Symbol string
}

// Currencies lists every currency the app accepts, in display order.
var Currencies = []Currency{
	{"EUR", "€"},
	{"USD", "$"},
	{"RUB", "₽"},
	{"GBP", "£"},
}

// CurrencySymbol returns the sign for a currency code, falling back to the code.
func CurrencySymbol(code string) string {
	for _, c := range Currencies {
		if c.Code == code {
			return c.Symbol
		}
	}
	return code
}

// IsCurrency reports whether code is one of the supported currencies.
func IsCurrency(code string) bool {
	for _, c := range Currencies {
		if c.Code == code {
			return true
		}
	}
	return false
}

// Validation errors returned by ParseAmount. Handlers map them to messages.
var (
	ErrAmountEmpty  = errors.New("amount is empty")
	ErrAmountShape  = errors.New("amount is not a number")
	ErrAmountZero   = errors.New("amount is zero or less")
	ErrAmountTooBig = errors.New("amount is too large")
)

// ParseAmount turns user input ("1250,50", "1 250.5", "38") into minor units.
// Both comma and dot are accepted as the decimal separator; spaces are ignored.
func ParseAmount(in string) (int64, error) {
	s := strings.TrimSpace(in)
	s = strings.NewReplacer(" ", "", nbsp, "", " ", "", "'", "").Replace(s)
	if s == "" {
		return 0, ErrAmountEmpty
	}
	s = strings.Replace(s, ",", ".", 1)
	if strings.ContainsAny(s, ",") {
		return 0, ErrAmountShape
	}

	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")
	s = strings.TrimPrefix(s, "-")

	whole, frac, hasFrac := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	if !isDigits(whole) || (hasFrac && !isDigits(frac)) {
		return 0, ErrAmountShape
	}
	if hasFrac && len(frac) > 2 {
		return 0, ErrAmountShape
	}
	if len(whole) > 10 {
		return 0, ErrAmountTooBig
	}

	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrAmountShape
	}
	var f int64
	switch len(frac) {
	case 0:
		f = 0
	case 1:
		f, _ = strconv.ParseInt(frac, 10, 64)
		f *= 10
	default:
		f, _ = strconv.ParseInt(frac, 10, 64)
	}

	total := w*minorUnits + f
	if neg || total <= 0 {
		return 0, ErrAmountZero
	}
	if total > MaxAmountMinor {
		return 0, ErrAmountTooBig
	}
	return total, nil
}

// groupDigits inserts a non-breaking space every three digits from the right.
func groupDigits(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	lead := n % 3
	if lead > 0 {
		b.WriteString(s[:lead])
	}
	for i := lead; i < n; i += 3 {
		if b.Len() > 0 {
			b.WriteString(nbsp)
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// MoneyParts splits an amount into the pieces the design renders at different
// sizes: the whole part, the decimal part (",50") and the currency sign.
type MoneyParts struct {
	Whole string
	Minor string
	Sign  string
	Fit   bool // true for long amounts: the hero style shrinks to 40px
}

// SplitMoney formats an amount for display. The number format is the same in
// both languages: "1 245,50 €".
func SplitMoney(minor int64, currency string) MoneyParts {
	if minor < 0 {
		minor = -minor
	}
	whole := groupDigits(strconv.FormatInt(minor/minorUnits, 10))
	frac := strconv.FormatInt(minor%minorUnits, 10)
	if len(frac) == 1 {
		frac = "0" + frac
	}
	return MoneyParts{
		Whole: whole,
		Minor: "," + frac,
		Sign:  nbsp + CurrencySymbol(currency),
		Fit:   len([]rune(whole)) > 7,
	}
}

// FormatMoney renders a plain one-line amount: "1 245,50 €".
func FormatMoney(minor int64, currency string) string {
	p := SplitMoney(minor, currency)
	return p.Whole + p.Minor + p.Sign
}

// FormatAmountInput renders an amount for a form field: "1250.50", no grouping,
// so the value round-trips through the input unchanged.
func FormatAmountInput(minor int64) string {
	p := SplitMoney(minor, "")
	return strings.ReplaceAll(p.Whole, nbsp, "") + p.Minor
}

// FormatCSVAmount renders an amount for CSV: "1250,50" without grouping.
func FormatCSVAmount(minor int64) string {
	return FormatAmountInput(minor)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
