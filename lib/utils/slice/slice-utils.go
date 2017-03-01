package slice

func DeleteFromInt(haystack []int64, needle int64) ([]int64, int64) {
	var numDeleted int64

	for idx, val := range haystack {
		if val == needle {
			haystack = append(haystack[:idx], haystack[idx+1:]...)
			numDeleted++
		}
	}

	return haystack, numDeleted
}

func GetIndexInt(haystack []int64, needle int64) int {
	for idx, val := range haystack {
		if val == needle {
			return idx
		}
	}

	return -1
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
