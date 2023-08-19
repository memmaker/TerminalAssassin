package mapset

import (
	"encoding/gob"
)

type MapSet[K comparable] map[K]bool

type Set[K comparable] interface {
	Add(v K) bool
	Contains(v K) bool
	ToSlice() []K
	Cardinality() int
	Remove(v K)
}

func NewSet[K comparable]() *MapSet[K] {
	m := make(MapSet[K])
	return &m
}

func (s *MapSet[K]) Iter(f func(K)) {
	for item := range *s {
		f(item)
	}
}

func (s *MapSet[K]) Remove(v K) {
	delete(*s, v)
}

func (s *MapSet[K]) Add(v K) bool {
	_, found := (*s)[v]
	(*s)[v] = true
	return !found //False if it existed already
}

func (s *MapSet[K]) Contains(v K) bool {
	_, found := (*s)[v]
	return found //true if it existed already
}

func (s *MapSet[K]) Cardinality() int {
	return len(*s)
}

func (s *MapSet[K]) Pop() (v K, ok error) {
	for item := range *s {
		delete(*s, item)
		return item, nil
	}
	return
}

func (s *MapSet[K]) ToSlice() []K {
	if s == nil {
		return []K{}
	}
	keys := make([]K, len(*s))
	i := 0
	for k := range *s {
		keys[i] = k
		i++
	}
	return keys
}

func (s *MapSet[K]) Clear() {
	for k := range *s {
		delete(*s, k)
	}
}

func (s *MapSet[K]) PeekOrDefault(defaultValue K) K {
	for k := range *s {
		return k
	}
	return defaultValue
}

func init() {
	gob.Register(&MapSet[string]{})
}
