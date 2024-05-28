package utils

func Find[T any, TS ~[]T](slice TS, predicate func(T) bool) (T, bool) {
	var def T
	for _, val := range slice {
		if predicate(val) {
			return val, true
		}
	}
	return def, false
}

func FindIndex[T any, TS ~[]T](slice TS, predicate func(T) bool) int {
	for i, val := range slice {
		if predicate(val) {
			return i
		}
	}
	return -1
}

func FindIndexByValue[T comparable, TS ~[]T](slice TS, predicate T) int {
	for i, val := range slice {
		if val == predicate {
			return i
		}
	}
	return -1
}
