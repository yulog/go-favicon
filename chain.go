package favicon

import "iter"

// MIT License
//
// Copyright (c) 2022 Tom Godkin
//
// Original:
// https://github.com/BooleanCat/go-functional/blob/48b826e9c890008826336c5e8ff2bc2a2f06c8b1/it/chain.go#L6C1-L12C2
func concat[T any](iterators ...func(func(T) bool)) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, iterator := range iterators {
			iterator(yield)
		}
	}
}
