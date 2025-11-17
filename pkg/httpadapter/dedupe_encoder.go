package httpadapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"unique"
)

// Keyer is implemented by values that expose a comparable key.
type Keyer[K comparable] interface {
	Key() K
}

// DedupByKey is a slice that dedupes itself based on a key during JSON unmarshal.
// The first instance of a key wins; ordering is preserved.
type DedupByKey[T Keyer[K], K comparable] []T

func (d *DedupByKey[T, K]) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	// Expect '['
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		return fmt.Errorf("DedupTodosUpdate: expected JSON array")
	}
	var decodeErr error
	dedupSeq := UniqueBy(decoderSeq[T](dec, &decodeErr), func(v T) K {
		return v.Key()
	})
	out := slices.Collect(dedupSeq)

	if decodeErr != nil {
		return decodeErr
	}

	// Expect closing ']'
	if tok, err = dec.Token(); err != nil {
		return err
	} else if delim, ok = tok.(json.Delim); !ok || delim != ']' {
		return fmt.Errorf("DedupTodosUpdate: expected closing ']'")
	}
	*d = out
	return nil
}

// UniqueBy returns an iterator that yields distinct items based on the provided key.
func UniqueBy[T any, K comparable](seq iter.Seq[T], key func(T) K) iter.Seq[T] {
	return func(yield func(T) bool) {
		seen := make(map[unique.Handle[K]]struct{})
		for v := range seq {
			h := unique.Make(key(v))
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			if !yield(v) {
				return
			}
		}
	}
}

func UniqueSeq[T comparable](seq iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		seen := make(map[unique.Handle[T]]struct{})
		for v := range seq {
			h := unique.Make(v)
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			if !yield(h.Value()) {
				return
			}
		}
	}
}

func UniqueSlice[T comparable](in []T) []T {
	return slices.Collect(UniqueSeq(slices.Values(in)))
}

func decoderSeq[T any](dec *json.Decoder, decodeErr *error) iter.Seq[T] {
	return func(yield func(T) bool) {
		count := 0
		for dec.More() {
			if count >= maxTodos {
				*decodeErr = fmt.Errorf("too many items (max %d)", maxTodos)
				return
			}
			var v T
			if err := dec.Decode(&v); err != nil {
				*decodeErr = err
				return
			}
			count++
			if !yield(v) {
				return
			}
		}
	}
}

const maxTodos = 5_00 // limit for the number of todos in a request
