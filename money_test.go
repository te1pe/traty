package main

import (
	"testing"
	"time"
)

func TestParseAmount(t *testing.T) {
	ok := []struct {
		in   string
		want int64
	}{
		{"1250,50", 125050},
		{"1250.50", 125050},
		{"1 250,50", 125050},
		{"38", 3800},
		{"0,01", 1},
		{"12,5", 1250},
		{"9999999,99", 999999999},
	}
	for _, c := range ok {
		got, err := ParseAmount(c.in)
		if err != nil {
			t.Errorf("ParseAmount(%q) returned error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseAmount(%q) = %d, want %d", c.in, got, c.want)
		}
	}

	bad := []struct {
		in   string
		want error
	}{
		{"", ErrAmountEmpty},
		{"   ", ErrAmountEmpty},
		{"0", ErrAmountZero},
		{"0,00", ErrAmountZero},
		{"-5", ErrAmountZero},
		{"abc", ErrAmountShape},
		{"12,345", ErrAmountShape},
		{"10000000", ErrAmountTooBig},
	}
	for _, c := range bad {
		if _, err := ParseAmount(c.in); err != c.want {
			t.Errorf("ParseAmount(%q) error = %v, want %v", c.in, err, c.want)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		minor int64
		cur   string
		want  string
	}{
		{125050, "EUR", "1 250,50 €"},
		{4550, "EUR", "45,50 €"},
		{0, "RUB", "0,00 ₽"},
		{100000000, "USD", "1 000 000,00 $"},
		{5, "GBP", "0,05 £"},
	}
	for _, c := range cases {
		if got := FormatMoney(c.minor, c.cur); got != c.want {
			t.Errorf("FormatMoney(%d, %s) = %q, want %q", c.minor, c.cur, got, c.want)
		}
	}
}

// A round trip through the form field must not change the amount.
func TestAmountRoundTrip(t *testing.T) {
	for _, minor := range []int64{1, 999, 125050, 999999999} {
		s := FormatAmountInput(minor)
		back, err := ParseAmount(s)
		if err != nil {
			t.Fatalf("ParseAmount(%q): %v", s, err)
		}
		if back != minor {
			t.Errorf("round trip %d -> %q -> %d", minor, s, back)
		}
	}
}

func TestWeekStart(t *testing.T) {
	day := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	}
	cases := []struct {
		in, want time.Time
	}{
		// Tuesday 22 Sept 2026 -> Monday 21 Sept
		{day(2026, 9, 22), day(2026, 9, 21)},
		// Monday stays itself
		{day(2026, 9, 21), day(2026, 9, 21)},
		// Sunday belongs to the week that started six days earlier
		{day(2026, 9, 27), day(2026, 9, 21)},
		// across a month boundary
		{day(2026, 10, 1), day(2026, 9, 28)},
	}
	for _, c := range cases {
		if got := WeekStart(c.in); !got.Equal(c.want) {
			t.Errorf("WeekStart(%s) = %s, want %s",
				FormatISO(c.in), FormatISO(got), FormatISO(c.want))
		}
	}
}

func TestPluralExpenses(t *testing.T) {
	cases := []struct {
		lang string
		n    int
		want string
	}{
		{LangRU, 1, "1 расход"},
		{LangRU, 2, "2 расхода"},
		{LangRU, 5, "5 расходов"},
		{LangRU, 11, "11 расходов"},
		{LangRU, 21, "21 расход"},
		{LangRU, 0, "0 расходов"},
		{LangEN, 1, "1 expense"},
		{LangEN, 3, "3 expenses"},
	}
	for _, c := range cases {
		if got := PluralExpenses(c.lang, c.n); got != c.want {
			t.Errorf("PluralExpenses(%s, %d) = %q, want %q", c.lang, c.n, got, c.want)
		}
	}
}
