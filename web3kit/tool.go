package web3kit

import (
	"bytes"
	binary "github.com/gagliardetto/binary"
	"reflect"
)

func Recover(err *error) {
	if r := recover(); r != nil {
		if e, ok := r.(error); ok {
			*err = e
		} else {
			panic(r)
		}
	}
}

func Map[T any, O any, TS ~[]T](input TS, mapper func(int, T) O) []O {
	var output = make([]O, len(input))
	for i, data := range input {
		output[i] = mapper(i, data)
	}
	return output
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}
func Must1[T any](arg T, err error) T {
	if err != nil {
		panic(err)
	}
	return arg
}
func Must2[T any, T2 any](arg T, arg2 T2, err error) (T, T2) {
	if err != nil {
		panic(err)
	}
	return arg, arg2
}
func Must3[T any, T2 any, T3 any](arg T, arg2 T2, arg3 T3, err error) (T, T2, T3) {
	if err != nil {
		panic(err)
	}
	return arg, arg2, arg3
}

func decode[T binary.BinaryUnmarshaler](data []byte, input T) error {
	var decoder = binary.NewDecoderWithEncoding(data, binary.EncodingBorsh)
	err := decoder.Decode(input)
	if err != nil {
		return err
	}
	return nil
}

func decodeObject[T binary.BinaryUnmarshaler](data []byte) (T, error) {
	var zero T
	if len(data) == 0 {
		return zero, nil
	}
	ret := reflect.New(reflect.TypeOf(zero).Elem()).Interface().(T)
	if err := decode(data, ret); err != nil {
		return zero, err
	}
	return ret, nil
}

func DeDupBy[T comparable, K comparable, TS ~[]T](input TS, selector func(T) K) []T {
	var m = make(map[K]bool, len(input))
	var n = make([]T, 0, len(m))
	for _, item := range input {
		key := selector(item)
		if _, ok := m[key]; !ok {
			m[key] = true
			n = append(n, item)
		}
	}
	return n
}

func GetSize(input binary.BinaryMarshaler) int {
	return len(GetBytes(input))
}

func GetBytes(input binary.BinaryMarshaler) []byte {
	var o = bytes.NewBuffer(nil)
	var encoder = binary.NewEncoderWithEncoding(o, binary.EncodingBorsh)
	err := input.MarshalWithEncoder(encoder)
	if err != nil {
		panic(err)
	}
	return o.Bytes()
}
