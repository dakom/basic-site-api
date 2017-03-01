package mathextra

import (
	"math"
	"strconv"
)

const LARGEST_BIT uint64 = 1 << (64 - 1)

func Round(num float64) int64 {
	return int64(num + math.Copysign(0.5, num))
}

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}

func GetRoundedPercentageAsString(total float64, partial float64) string {
	return strconv.FormatInt(Round((partial / total * 100)), 10)
}

func MaxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func MaxUInt64(a uint64, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func MinInt64(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MinUInt64(a uint64, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
