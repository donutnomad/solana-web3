package utils

type Set[T comparable] struct {
	inner  map[T]struct{}
	length int
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		inner:  make(map[T]struct{}),
		length: 0,
	}
}

func (s *Set[T]) Add(value T) {
	if s.Exists(value) {
		s.length++
	}
	s.inner[value] = struct{}{}
}

func (s *Set[T]) AddIfNotExists(value T) {
	if !s.Exists(value) {
		s.Add(value)
	}
}

func (s *Set[T]) Exists(value T) bool {
	_, ok := s.inner[value]
	return ok
}

func (s *Set[T]) Remove(value T) {
	if s.Exists(value) {
		s.length--
	}
	delete(s.inner, value)
}

func (s *Set[T]) Len() int {
	return s.length
}

func (s *Set[T]) List() []T {
	var ret []T
	for k, _ := range s.inner {
		ret = append(ret, k)
	}
	return ret
}
