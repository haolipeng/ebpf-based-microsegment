// input: elements to add/remove from set
// output: set membership queries, set operations
// pos: graph - generic set utility for graph operations
//
// Package graph provides an in-memory graph database implementation.
// This Set implementation is adapted from github.com/deckarep/golang-set (MIT License).

package graph

// Set is a thread-unsafe set implementation.
type Set map[interface{}]struct{}

// NewSet creates and returns a new empty Set.
func NewSet() Set {
	return make(Set)
}

// Add adds an element to the set. Returns true if the item was added.
func (s Set) Add(i interface{}) bool {
	if _, ok := s[i]; ok {
		return false
	}
	s[i] = struct{}{}
	return true
}

// Cardinality returns the number of elements in the set.
func (s Set) Cardinality() int {
	return len(s)
}

// Clear removes all elements from the set.
func (s Set) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// Clone returns a shallow copy of the set.
func (s Set) Clone() Set {
	clone := NewSet()
	for k := range s {
		clone.Add(k)
	}
	return clone
}

// Contains returns true if all given items are in the set.
func (s Set) Contains(items ...interface{}) bool {
	for _, item := range items {
		if _, ok := s[item]; !ok {
			return false
		}
	}
	return true
}

// Difference returns a new set with elements in this set but not in other.
func (s Set) Difference(other Set) Set {
	result := NewSet()
	for k := range s {
		if !other.Contains(k) {
			result.Add(k)
		}
	}
	return result
}

// Equal returns true if both sets contain the same elements.
func (s Set) Equal(other Set) bool {
	if s.Cardinality() != other.Cardinality() {
		return false
	}
	for k := range s {
		if !other.Contains(k) {
			return false
		}
	}
	return true
}

// Intersect returns a new set with elements common to both sets.
func (s Set) Intersect(other Set) Set {
	result := NewSet()
	// Iterate over the smaller set for efficiency
	if s.Cardinality() < other.Cardinality() {
		for k := range s {
			if other.Contains(k) {
				result.Add(k)
			}
		}
	} else {
		for k := range other {
			if s.Contains(k) {
				result.Add(k)
			}
		}
	}
	return result
}

// IsSubset returns true if every element in this set is in other.
func (s Set) IsSubset(other Set) bool {
	for k := range s {
		if !other.Contains(k) {
			return false
		}
	}
	return true
}

// IsSuperset returns true if every element in other is in this set.
func (s Set) IsSuperset(other Set) bool {
	return other.IsSubset(s)
}

// Remove removes an element from the set.
func (s Set) Remove(i interface{}) {
	delete(s, i)
}

// Union returns a new set with elements from both sets.
func (s Set) Union(other Set) Set {
	result := s.Clone()
	for k := range other {
		result.Add(k)
	}
	return result
}

// Iter returns a channel for iterating over set elements.
func (s Set) Iter() <-chan interface{} {
	ch := make(chan interface{})
	go func() {
		for k := range s {
			ch <- k
		}
		close(ch)
	}()
	return ch
}

// ToSlice returns all elements as a slice.
func (s Set) ToSlice() []interface{} {
	result := make([]interface{}, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	return result
}

// StringSet is a type-safe set for strings.
type StringSet map[string]struct{}

// NewStringSet creates a new empty string set.
func NewStringSet() StringSet {
	return make(StringSet)
}

// Add adds a string to the set.
func (s StringSet) Add(str string) bool {
	if _, ok := s[str]; ok {
		return false
	}
	s[str] = struct{}{}
	return true
}

// Contains returns true if the string is in the set.
func (s StringSet) Contains(str string) bool {
	_, ok := s[str]
	return ok
}

// Remove removes a string from the set.
func (s StringSet) Remove(str string) {
	delete(s, str)
}

// Cardinality returns the number of elements.
func (s StringSet) Cardinality() int {
	return len(s)
}

// ToSlice returns all strings as a slice.
func (s StringSet) ToSlice() []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	return result
}
