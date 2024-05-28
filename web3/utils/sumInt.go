package utils

func SumInt[T uint8 | int8 | uint16 | int16 | uint32 | int32 | uint64 | int64 | int | uint, O any](base []O, callback func(O) T) T {
	var ret T = 0
	for _, item := range base {
		ret += callback(item)
	}
	return ret
}
