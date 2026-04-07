package goredditads

import (
	"math"

	"github.com/ndx-technologies/fmtx"
)

// CTRSigStr formats a CTR z-score as a significance string with stars.
func CTRSigStr(z float64, ok bool) string {
	if !ok {
		return fmtx.DimS("-")
	}
	absZ := math.Abs(z)
	switch {
	case absZ > 3.29:
		if z > 0 {
			return fmtx.GreenS("★★★")
		}
		return fmtx.RedS("▼★★★")
	case absZ > 2.58:
		if z > 0 {
			return fmtx.GreenS("★★")
		}
		return fmtx.RedS("▼★★")
	case absZ > 1.96:
		if z > 0 {
			return fmtx.GreenS("★")
		}
		return fmtx.RedS("▼★")
	default:
		return fmtx.DimS("-")
	}
}
