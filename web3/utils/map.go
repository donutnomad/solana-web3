package utils

type KVMap[K comparable, V any] struct {
	inner map[K]V
}

func NewKVMap[K comparable, V any]() *KVMap[K, V] {
	return &KVMap[K, V]{
		inner: make(map[K]V),
	}
}

func (m *KVMap[K, V]) InsertOrUpdate(key K, value V, update func(*V)) {
	if v, ok := m.inner[key]; ok {
		update(&v)
		m.inner[key] = v
	} else {
		m.inner[key] = value
	}
}

func (m *KVMap[K, V]) Entries() (keys []K, values []V) {
	for k, v := range m.inner {
		keys = append(keys, k)
		values = append(values, v)
	}
	return
}

func (m *KVMap[K, V]) Keys() (out []K) {
	for k := range m.inner {
		out = append(out, k)
	}
	return
}

func (m *KVMap[K, V]) Values() (out []V) {
	for _, v := range m.inner {
		out = append(out, v)
	}
	return
}
