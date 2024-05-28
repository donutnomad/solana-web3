package utils

func PowInt[T uint8 | int8 | uint16 | int16 | uint32 | int32 | uint64 | int64 | int | uint](base T, p int) T {
	var ret T = 1
	for i := 0; i < p; i++ {
		ret *= base
	}
	return ret
}
