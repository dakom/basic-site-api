package text

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func RandomHexString(byteLength uint64) (string, error) {
	bytes := make([]byte, byteLength)
	_, err := rand.Read(bytes)

	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func StringToInt64(str string) (int64, error) {
	if len(str) < 1 {
		return 0, fmt.Errorf("NOEXIST")
	}

	ret, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

func MakeJsonString(jsonMap map[string]interface{}) (string, error) {
	if jsonMap == nil {
		return "", nil
	}

	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		return "", err
	}
	jsonString := string(jsonBytes)

	return jsonString, nil
}

func ClearCommas(str string) string {
	str = strings.Replace(str, ",", " ", -1)
	str = strings.Replace(str, "  ", " ", -1)
	return str
}
