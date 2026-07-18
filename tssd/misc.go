package tssd

import (
	"fmt"
)

func SliceEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func SliceSliceEqual[T comparable](a, b [][]T) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !SliceEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

type Equaler interface {
	Equal(Equaler) bool
}

func MapEqual[K comparable, T Equaler](a, b map[K]T) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if !v.Equal(b[k]) {
			return false
		}
	}
	return true
}

// convert a sender buffer to a receiver buffer
// normall it is called in test only
func Pipe(sender *Buffer) (receiver *Buffer) {

	receiver = &Buffer{}

	//TSSD produce in the sender.FragmentData
	for i := 0; i < len(sender.Fragments); i++ {
		frag := &Fragment{}
		_, err := frag.Unmarshal(sender.Fragments[i].Raw)
		if err != nil {
			fmt.Println("data:", sender.Fragments[i].Raw)
			panic("pipe output unmashal fail")
		}
		receiver.Push(frag)
	}

	return receiver

}
