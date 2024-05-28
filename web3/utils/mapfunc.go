package utils

import "golang.org/x/exp/constraints"

func Map[T any, O any, TS ~[]T](input TS, mapper func(T) O) []O {
	var newDatas = make([]O, 0, len(input))
	for _, data := range input {
		newDatas = append(newDatas, mapper(data))
	}
	return newDatas
}

func MapWithError[T any, O any, TS ~[]T](input TS, mapper func(T) (O, error)) ([]O, error) {
	var newDatas = make([]O, 0, len(input))
	for _, data := range input {
		ret, err := mapper(data)
		if err != nil {
			return nil, err
		}
		newDatas = append(newDatas, ret)
	}
	return newDatas, nil
}

func MapInt[T constraints.Integer, O constraints.Integer, TS ~[]T](input TS) []O {
	var newDatas = make([]O, 0, len(input))
	for _, data := range input {
		newDatas = append(newDatas, O(data))
	}
	return newDatas
}

func Contain[T comparable](slice []T, target T) bool {
	for _, data := range slice {
		if data == target {
			return true
		}
	}
	return false
}

func DeDup[T comparable, TS ~[]T](input TS) []T {
	var m = make(map[T]bool, len(input))
	var n = make([]T, 0, len(m))
	for _, item := range input {
		if _, ok := m[item]; !ok {
			m[item] = true
			n = append(n, item)
		}
	}

	return n
}

func AppendToFirst[T any](arr []T, ele T) []T {
	return append([]T{ele}, arr...)
}

func RemoveEle[T comparable](arr []T, ele func(T) bool) []T {
	var ret []T
	for _, item := range arr {
		if !ele(item) {
			ret = append(ret, item)
		}
	}
	return ret
}

func RemoveIndex[T any](arr []T, index int) []T {
	var ret []T
	for idx, item := range arr {
		if idx != index {
			ret = append(ret, item)
		}
	}
	return ret
}

func MergeList[T any, TS ~[]T](args ...TS) []T {
	var count int
	for _, arg := range args {
		count += len(arg)
	}
	var ret = make([]T, 0, count)
	for _, arg := range args {
		ret = append(ret, arg...)
	}
	return ret
}

func DeRef2[T any](input *T, err error) (T, error) {
	return *input, err
}
