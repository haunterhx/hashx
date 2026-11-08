package entropy

import (
	"math"
)

func Shannon(hist [256]uint64, total uint64) float64 {
	if total == 0 {
		return 0
	}

	var e float64
	for _, c := range hist {
		if c == 0 {
			continue
		}
		p := float64(c) / float64(total)
		e -= p * math.Log2(p)
	}
	return e
}
