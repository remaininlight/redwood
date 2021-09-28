package utils

import (
	"redwood.dev/types"
)

type HashSet map[types.Hash]struct{}

func NewHashSet(vals []types.Hash) HashSet {
	set := map[types.Hash]struct{}{}
	for _, val := range vals {
		set[val] = struct{}{}
	}
	return set
}

func (s HashSet) Add(val types.Hash) HashSet {
	s[val] = struct{}{}
	return s
}

func (s HashSet) Remove(val types.Hash) HashSet {
	delete(s, val)
	return s
}

func (s HashSet) Any() types.Hash {
	for x := range s {
		return x
	}
	return types.Hash{}
}

func (s HashSet) Slice() []types.Hash {
	var slice []types.Hash
	for x := range s {
		slice = append(slice, x)
	}
	return slice
}

func (s HashSet) Copy() HashSet {
	set := map[types.Hash]struct{}{}
	for val := range s {
		set[val] = struct{}{}
	}
	return set
}
