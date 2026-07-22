package tssd

import (
	"fmt"
	"math/rand"
	"time"
)

// put a binary content into a slice
// yeah! convert everything to byte slice
func Slice(k Ptr, size Size_t) []byte {
	p := (*[1<<31 - 1]byte)(k) //yeah, it's magic number, which is maxnum can be accept by golang compiler
	return (*p)[0:size]
}

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
	numbers := make([]int, len(sender.Fragments))
	for i := 0; i < len(numbers); i++ {
		numbers[i] = i
	}
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
	// Shuffle the slice
	rand.Shuffle(len(numbers), func(i, j int) {
		numbers[i], numbers[j] = numbers[j], numbers[i]
	})

	//TSSD produce in the sender.FragmentData
	for i := 0; i < len(sender.Fragments); i++ {
		frag := &Fragment{}
		_, err := frag.Unmarshal(sender.Fragments[numbers[i]].Data)
		if err != nil {
			fmt.Println("data:", sender.Fragments[numbers[i]].Data, numbers[i], err)
			panic("pipe output unmashal fail")
		}
		receiver.Push(frag)
	}
	return receiver
}

func randBytes(n int) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ret := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, uint8(r.Intn(255)))
	}
	return ret
}

const (
	minUint8 = uint8(0)
	maxUint8 = ^uint8(0)

	minUint16 = uint16(0)
	maxUint16 = ^uint16(0)

	minUint32 = uint32(0)
	maxUint32 = ^uint32(0)

	minUint64 = uint64(0)
	maxUint64 = ^uint64(0)

	minInt32 = int32(-maxInt32 - 1)
	maxInt32 = int32(maxUint32 >> 1)

	minInt64 = int64(-maxInt64 - 1)
	maxInt64 = int64(maxUint64 >> 1)
)
