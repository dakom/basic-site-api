package slice

func DeleteFromString(haystack []string, needle string) ([]string, int64) {
	var numDeleted int64

	for idx := 0; idx < len(haystack); idx++ {
		val := haystack[idx]
		if val == needle {
			haystack = append(haystack[:idx], haystack[idx+1:]...)
			numDeleted++
			idx-- //need to go back an index since we've modified it
		}
	}

	return haystack, numDeleted
}

func DeleteFromInt(haystack []int64, needle int64) ([]int64, int64) {
	var numDeleted int64

	for idx := 0; idx < len(haystack); idx++ {
		val := haystack[idx]
		if val == needle {
			haystack = append(haystack[:idx], haystack[idx+1:]...)
			numDeleted++
			idx-- //need to go back an index since we've modified it
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
