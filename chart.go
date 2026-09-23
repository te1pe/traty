package main

import "math"

// Donut geometry from the design system: viewBox 136x136, radius 58, stroke 12.
const (
	donutRadius = 58.0
	donutGap    = 2.5 // gap between segments, in path units
	donutMinSeg = 0.8 // a tiny category still gets a visible tick
)

// donutCircumference is the full length of the ring path.
var donutCircumference = 2 * math.Pi * donutRadius

// maxDonutSegments caps the ring at five arcs; beyond that the tail is folded
// into a neutral "Other" segment, as the design requires.
const maxDonutSegments = 5

// BuildDonut takes category totals (already sorted by amount, largest first)
// and returns the segments to draw: percentages plus the dash geometry each
// <circle> needs. Everything is computed on the server so the page needs no JS.
func BuildDonut(totals []CategoryTotal, total int64, otherLabel string) []CategoryTotal {
	if total <= 0 || len(totals) == 0 {
		return nil
	}

	segments := make([]CategoryTotal, 0, maxDonutSegments)
	if len(totals) > maxDonutSegments {
		segments = append(segments, totals[:maxDonutSegments-1]...)
		other := CategoryTotal{Name: otherLabel, IsOther: true}
		for _, t := range totals[maxDonutSegments-1:] {
			other.TotalMinor += t.TotalMinor
			other.Count += t.Count
		}
		segments = append(segments, other)
	} else {
		segments = append(segments, totals...)
	}

	start := 0.0
	for i := range segments {
		share := float64(segments[i].TotalMinor) / float64(total)
		arc := share * donutCircumference

		segments[i].Percent = share * 100
		segments[i].DashLength = math.Max(arc-donutGap, donutMinSeg)
		segments[i].DashOffset = -(start + donutGap/2)
		start += arc
	}
	return segments
}

// Gap is the rest of the ring: the second value of stroke-dasharray.
func (t CategoryTotal) Gap() float64 { return donutCircumference - t.DashLength }

// WithPercent fills in each row's share of the total (the number next to the
// amount) and the width of its bar, which the design measures against the
// largest category so the biggest bar is always full.
func WithPercent(totals []CategoryTotal, total int64) []CategoryTotal {
	if total <= 0 {
		return totals
	}
	max := MaxTotal(totals)
	for i := range totals {
		totals[i].Percent = float64(totals[i].TotalMinor) / float64(total) * 100
		if max > 0 {
			totals[i].BarPercent = float64(totals[i].TotalMinor) / float64(max) * 100
		}
	}
	return totals
}

// MaxTotal returns the largest amount in a breakdown, used to scale the bars.
func MaxTotal(totals []CategoryTotal) int64 {
	var max int64
	for _, t := range totals {
		if t.TotalMinor > max {
			max = t.TotalMinor
		}
	}
	return max
}
