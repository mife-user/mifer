package exc

import (
	"strconv"
)

// string to uint
func StrToUint(str string) (uint, error) {
	idUint, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(idUint), nil
}

// uint to string
func UintToStr(id uint) (string, error) {
	return strconv.FormatUint(uint64(id), 10), nil
}
