package main

import (
	"fmt"
	"strings"
	"time"
)

// dateLayout is the single date format used in URLs, forms, the database and CSV.
const dateLayout = "2006-01-02"

// Today returns the current local date, truncated to midnight in the server's
// timezone. Everything in the app — "today", "this week" — is a local calendar
// date, never UTC, so the numbers are right just after midnight too.
func Today() time.Time {
	return DayOf(time.Now())
}

// DayOf drops the time part, keeping the location.
func DayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// WeekStart returns the Monday of the week that contains d.
// Sunday belongs to the week that started six days earlier.
func WeekStart(d time.Time) time.Time {
	d = DayOf(d)
	offset := (int(d.Weekday()) + 6) % 7 // Monday -> 0, Sunday -> 6
	return d.AddDate(0, 0, -offset)
}

// MonthStart returns the first day of d's month.
func MonthStart(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
}

// ParseDate reads a YYYY-MM-DD string as a local date.
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, strings.TrimSpace(s), time.Local)
}

// FormatISO renders a date as YYYY-MM-DD.
func FormatISO(d time.Time) string { return d.Format(dateLayout) }

var monthsGenitiveRU = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря"}

var monthsShortRU = [...]string{"янв.", "февр.", "мар.", "апр.", "мая", "июня",
	"июля", "авг.", "сент.", "окт.", "нояб.", "дек."}

var monthsEN = [...]string{"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

var monthsShortEN = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sept", "Oct", "Nov", "Dec"}

var weekdaysRU = [...]string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

var weekdaysShortRU = [...]string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}

var weekdaysEN = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

var weekdaysShortEN = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// FormatDayLong renders "22 сентября" / "22 September".
func FormatDayLong(d time.Time, lang string) string {
	m := int(d.Month()) - 1
	if lang == LangEN {
		return fmt.Sprintf("%d %s", d.Day(), monthsEN[m])
	}
	return fmt.Sprintf("%d %s", d.Day(), monthsGenitiveRU[m])
}

// FormatWeekdayDay renders "вторник, 22 сентября" / "Tuesday 22 September".
func FormatWeekdayDay(d time.Time, lang string) string {
	w := int(d.Weekday())
	if lang == LangEN {
		return fmt.Sprintf("%s %s", weekdaysEN[w], FormatDayLong(d, lang))
	}
	return fmt.Sprintf("%s, %s", weekdaysRU[w], FormatDayLong(d, lang))
}

// FormatDayShort renders "22 сент." / "22 Sept", adding the year when the date
// is not in the current year.
func FormatDayShort(d time.Time, lang string) string {
	m := int(d.Month()) - 1
	sameYear := d.Year() == time.Now().Year()
	if lang == LangEN {
		s := fmt.Sprintf("%d %s", d.Day(), monthsShortEN[m])
		if !sameYear {
			s += fmt.Sprintf(" %d", d.Year())
		}
		return s
	}
	s := fmt.Sprintf("%d %s", d.Day(), monthsShortRU[m])
	if !sameYear {
		s += fmt.Sprintf(" %d г.", d.Year())
	}
	return s
}

// FormatDayRelative renders "Сегодня" / "Вчера" and falls back to a short date.
func FormatDayRelative(d, today time.Time, lang string) string {
	switch {
	case d.Equal(today):
		return T(lang, "date.today")
	case d.Equal(today.AddDate(0, 0, -1)):
		return T(lang, "date.yesterday")
	}
	return FormatDayShort(d, lang)
}

// FormatWeekdaySpan renders the days a week card covers: "пн — вт" / "Mon – Tue".
func FormatWeekdaySpan(from, to time.Time, lang string) string {
	short := weekdaysShortRU
	dash := " — "
	if lang == LangEN {
		short = weekdaysShortEN
		dash = " – "
	}
	a, b := short[int(from.Weekday())], short[int(to.Weekday())]
	if a == b {
		return a
	}
	return a + dash + b
}

// FormatRange renders a report period: "1–22 сентября 2026 г." / "1 – 22 September 2026".
func FormatRange(from, to time.Time, lang string) string {
	en := lang == LangEN
	yearRU := func(y int) string { return fmt.Sprintf(" %d г.", y) }

	switch {
	case from.Equal(to):
		if en {
			return fmt.Sprintf("%d %s %d", from.Day(), monthsEN[from.Month()-1], from.Year())
		}
		return fmt.Sprintf("%d %s%s", from.Day(), monthsGenitiveRU[from.Month()-1], yearRU(from.Year()))

	case from.Year() == to.Year() && from.Month() == to.Month():
		if en {
			return fmt.Sprintf("%d – %d %s %d", from.Day(), to.Day(), monthsEN[to.Month()-1], to.Year())
		}
		return fmt.Sprintf("%d–%d %s%s", from.Day(), to.Day(), monthsGenitiveRU[to.Month()-1], yearRU(to.Year()))

	case from.Year() == to.Year():
		if en {
			return fmt.Sprintf("%d %s – %d %s %d", from.Day(), monthsEN[from.Month()-1],
				to.Day(), monthsEN[to.Month()-1], to.Year())
		}
		return fmt.Sprintf("%d %s – %d %s%s", from.Day(), monthsGenitiveRU[from.Month()-1],
			to.Day(), monthsGenitiveRU[to.Month()-1], yearRU(to.Year()))
	}

	if en {
		return fmt.Sprintf("%d %s %d – %d %s %d", from.Day(), monthsEN[from.Month()-1], from.Year(),
			to.Day(), monthsEN[to.Month()-1], to.Year())
	}
	return fmt.Sprintf("%d %s%s – %d %s%s", from.Day(), monthsGenitiveRU[from.Month()-1], yearRU(from.Year()),
		to.Day(), monthsGenitiveRU[to.Month()-1], yearRU(to.Year()))
}
