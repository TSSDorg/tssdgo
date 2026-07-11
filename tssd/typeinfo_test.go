package tssd

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
	"unsafe"
	//"strconv"
	//"assert"
	//tssd "github.com/simpleKV/tssd/tssd"
)

type S1 struct {
	T   time.Time
	V   int16
	F   float32
	Arr [2]uint64
}

type TestStruct struct {
	V int
	T time.Time
	S1
	V2 uint8
}

func TestString(t *testing.T) {

	type ss struct {
		I   int
		Str string
		I16 int16
	}
	sin1 := ss{
		10,
		"",
		21,
	}
	sin2 := ss{
		10,
		"afsdfsfsdfgdsgfdfgdrgrgeertgr",
		21,
	}
	var s2 ss
	container := parse(sin1)

	b, e := container.marshal(&sin1, make([]byte, 0, 2048))
	if e != nil || len(b) == 0 {
		t.Errorf("Test String Marshal err %s", e)
	}

	fmt.Println("testString out buf:", b)
	container.print(b)
	fmt.Println("testString out end")

	container.unmarshal(b, &s2)
	if !reflect.DeepEqual(sin1, s2) {
		t.Errorf("Test String err: [%s], [%s]", sin1.Str, s2.Str)
	}

	b, e = container.marshal(&sin2, make([]byte, 0, 2048))
	if e != nil {
		t.Errorf("Test String Marshal err %s", e)
	}
	container.print(b)
	container.unmarshal(b, &s2)
	if !reflect.DeepEqual(sin2, s2) {
		t.Errorf("Test String err: [%s], [%s]", sin2.Str, s2.Str)
	}
}

func TestSimpleStringSlice(t *testing.T) {

	type slice struct {
		Ss []string
	}

	sin := slice{
		[]string{"a", "b"},
	}
	var sout slice
	container := parse(sin)
	buf, _ := container.marshal(&sin, make([]byte, 0, 2048))
	container.print(buf)
	fmt.Println("TestSimpleStringSlice buf:", buf)

	container.unmarshal(buf, &sout)
	if !reflect.DeepEqual(sin, sout) {
		fmt.Println("TestSimpleStringSlice sout:", sout)
		t.Errorf("Test String slice err ")
	}
}

func TestStringSlice(t *testing.T) {

	type ss struct {
		I    int
		Strs []string
		I16  int16
	}
	sin1 := ss{
		10,
		[]string{"", "abc", "", "a"},
		21,
	}
	var s2 ss
	container := parse(sin1)
	buf, _ := container.marshal(&sin1, make([]byte, 0, 2048))

	container.unmarshal(buf, &s2)
	if !reflect.DeepEqual(sin1, s2) {
		t.Errorf("Test String slice err ")
	}
}

func TestStringArray(t *testing.T) {

	type ss struct {
		I    int
		Strs [4]string
		I16  int16
	}
	sin1 := ss{
		10,
		[4]string{"", "abc", "", "a"},
		21,
	}
	var s2 ss
	container := parse(sin1)
	n, _ := container.marshal(&sin1, make([]byte, 0, 2048))

	fmt.Println(container)
	fmt.Println(sin1, n)

	container.unmarshal(n, &s2)
	if !reflect.DeepEqual(sin1, s2) {
		t.Errorf("Test String array err")
	}
}

func TestSliceXXX(t *testing.T) {

	type ss struct {
		I    []uint
		Strs []string
		I64  []int64
		B    []byte
	}
	in1 := ss{
		[]uint{10, 123},
		[]string{"abc"},
		[]int64{1, 0},
		[]byte("hello world"),
	}
	var s2 ss

	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))
	fmt.Println("=========================TestSlice================================", n)
	fmt.Printf("%p, %p\n", &s2, &s2.I)
	fmt.Println(s2)
	//p := (*[]byte)(Ptr(&s2))
	//*p = make([]byte, 16)
	//*p = (*p)[0:2] //set size

	fmt.Println("after alloc:", s2)

	container.unmarshal(n, &s2)
	fmt.Printf("after unmarshal %p, %p %p %p %p\n", &s2, &s2.I, &s2.I[0], &s2.I64, &s2.I64[0])
	if !reflect.DeepEqual(in1, s2) {
		fmt.Println(in1, s2)
		t.Errorf("Test String slice err ")
	}
}

func TestSliceUint(t *testing.T) {

	in1 := []int8{1, 2, 3, 5, 4}

	var s2 []int8

	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))
	fmt.Println(s2)

	container.unmarshal(n, &s2)
	fmt.Println("after alloc:", s2)
	//fmt.Printf("after unmarshal %p, %p %p\n", &s2, &s2.I, &s2.I[0])
	if !reflect.DeepEqual(in1, s2) {
		fmt.Println(in1, s2)
		t.Errorf("Test String slice err ")
	}
}

func TestSlice(t *testing.T) {

	type ss struct {
		I    []int
		Strs []string
		I64  []int64
		B    []byte
	}
	in1 := ss{
		[]int{10, 123},
		[]string{"", "abc", "", "a"},
		[]int64{1, 0, 3456789},
		[]byte("hello world"),
	}
	var s2 ss

	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))
	fmt.Println("=========================TestSlice================================", n)

	container.unmarshal(n, &s2)
	if !reflect.DeepEqual(in1, s2) {
		fmt.Println(in1, s2)
		t.Errorf("Test String slice err ")
	}

	in1 = ss{
		[]int{},
		[]string{},
		[]int64{},
		[]byte{},
	}
	n, _ = container.marshal(&in1, make([]byte, 0, 2048))

	container.unmarshal(n, &s2)
	fmt.Println("Test slice:", len(s2.I), len(s2.Strs), len(s2.I64), len(s2.B))
	if len(s2.I) > 0 || len(s2.Strs) > 0 || len(s2.I64) > 0 || len(s2.B) > 0 {
		t.Errorf("Test slice err:")
	}
}

func TestArray(t *testing.T) {

	type ss struct {
		I    [2]int
		Strs [4]string
		I64  [4]int64
		B    [5]byte
	}
	in1 := ss{
		[2]int{10, 123},
		[4]string{"abc", "", "abcd", "a"},
		[4]int64{13445, 0, 3456789, -23435345},
		[5]byte{'h', 'e', 'l', 'l', '0'},
	}
	var s2 ss
	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))
	fmt.Println(in1)

	container.unmarshal(n, &s2)
	if !reflect.DeepEqual(in1, s2) {
		fmt.Println(s2)
		t.Errorf("Test array err ")
	}
}

func TestNestStruct(t *testing.T) {

	type s struct {
		I   int
		Str string
		Ss  []string
	}

	type ss struct {
		I    [2]int
		Nest s
		Strs []string
	}
	in1 := ss{
		[2]int{10, 123},
		s{456, "hello", []string{"", "abcd", "a"}},
		[]string{"abc", "", "abcd", "a"},
	}
	var s2 ss
	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))
	fmt.Println(in1)

	container.unmarshal(n, &s2)
	if !reflect.DeepEqual(in1, s2) {
		fmt.Println(s2)
		t.Errorf("Test TestNestStruct err ")
	}
}

func TestNestStructSlice(t *testing.T) {

	type sin struct {
		I int
		//str string
		Ss []string
	}

	type sout struct {
		I    []int
		Nest []sin
		//strs []string
	}
	in1 := sout{
		[]int{10, 123},
		[]sin{{123, []string{""}}, {2, []string{"", "abc"}}, {0, []string{"", ""}}, {234, []string{"abc", ""}}},
		//[]string{"abc", "", "abcd", "a"},
	}
	fmt.Println("Test TestNestStructSlice begin ~~~~~~~~~~~~~~")
	var s2 sout
	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))
	fmt.Println(in1)

	container.unmarshal(n, &s2)
	if !reflect.DeepEqual(in1, s2) {
		fmt.Println(s2, len(s2.Nest))
		t.Errorf("Test TestNestStructSlice err ")
	}
}

func TestEmptySlice(t *testing.T) {

	type sin struct {
		//str string
		Ss []string
	}

	in1 := sin{
		[]string{},
		//[2]sin{{1, "hello"}, {2, ""}},
		//[4]sin{{123, []string{}}, {2, []string{"", "abc"}}, {0, []string{"", ""}}, {234, []string{"abc", ""}}},
		//[]string{"abc", "", "abcd", "a"},
	}
	fmt.Println("Test TestEmptySlice begin ~~~~~~~~~~~~~~")
	var s2 sin
	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))

	container.unmarshal(n, &s2)
	fmt.Println("in1:", in1)
	fmt.Println("out:", s2)
	//TODO
	if len(s2.Ss) != 0 {
		t.Errorf("Test TestEmptySlice err ")
	}
}

func TestNestStructArray(t *testing.T) {

	type sin struct {
		I int
		//str string
		Ss []string
	}

	type sout struct {
		I    []int
		Nest [4]sin
		//strs []string
	}
	in1 := sout{
		[]int{10, 123},
		//[2]sin{{1, "hello"}, {2, ""}},
		[4]sin{{123, []string{""}}, {2, []string{"", "abc"}}, {0, []string{"", ""}}, {234, []string{"abc", ""}}},
		//[]string{"abc", "", "abcd", "a"},
	}
	fmt.Println("Test TestNestStructArray begin ~~~~~~~~~~~~~~")
	var s2 sout
	container := parse(in1)
	n, _ := container.marshal(&in1, make([]byte, 0, 2048))
	fmt.Println("in1:", in1)

	container.unmarshal(n, &s2)
	if !reflect.DeepEqual(in1, s2) {
		fmt.Println("out:", s2)
		t.Errorf("Test TestNestStructArray err ")
	}
}

func TestMap(t *testing.T) {
	type st1 struct {
		M1 map[int]string
		S  string
		M2 map[string]string
	}
	type st struct {
		I  int
		M  map[string]int
		Is []uint16
		St st1
		M2 map[string]string
	}

	var s1, s2 st
	s1.I = 12
	s1.M = make(map[string]int, 0)
	s1.M["hello"] = 21
	s1.M["world"] = 156
	s1.Is = append(s1.Is, 31)
	s1.Is = append(s1.Is, 43)

	s1.M2 = make(map[string]string, 0)
	s1.M2["hsf"] = "sfeefer"
	s1.M2["sfesfe"] = "weereee"
	s1.M2["sfesf2e"] = ""
	s1.M2[""] = ""

	s1.St.M1 = make(map[int]string, 0)
	s1.St.M1[2] = "heeee2"
	s1.St.M1[8] = "hhh8"
	s1.St.S = "sfeerfer"

	s1.St.M2 = make(map[string]string, 0)
	s1.St.M2["wee"] = "wefefe"
	s1.St.M2["wee2"] = "wefefe2"
	s1.St.M2["we"] = "wefwereeefe"

	container := parse(s1)
	b, _ := container.marshal(&s1, make([]byte, 0, 2048))

	container.unmarshal(b, &s2)

	fmt.Println("s2", s2)

	//s2.i = 13

	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestMap failed")
	}
}

func TestSimpleTime(t *testing.T) {

	tt := time.Now()
	container := parse(tt)
	b, _ := container.marshal(&tt, make([]byte, 0, 2048))

	var v2 time.Time
	container.unmarshal(b, &v2)
	fmt.Println("tt:", tt.Format(time.RFC3339Nano))
	fmt.Println("v2:", v2.Format(time.RFC3339Nano))
	if !v2.Equal(tt) {
		t.Error("TestSimpleTime failed")
	}
}

func TestEmbedStruct(t *testing.T) {

	v := TestStruct{V: 2, T: time.Now()}
	v.S1.V = 3

	fmt.Println(v)
	container := parse(v)
	b, _ := container.marshal(&v, make([]byte, 0, 2048))

	var v2 TestStruct
	container.unmarshal(b, &v2)
	if !v.T.Equal(v2.T) || v.S1.V != v2.S1.V {
		t.Error("TestTime failed")
	}
}

func TestTime(t *testing.T) {

	v := TestStruct{V: 2, T: time.Now()}
	v.S1.V = 3

	fmt.Println(v)
	container := parse(v)
	b, _ := container.marshal(&v, make([]byte, 0, 2048))

	var v2 TestStruct
	container.unmarshal(b, &v2)
	if !v.T.Equal(v2.T) || v.S1.V != v2.S1.V {
		t.Error("TestTime failed")
	}
}

func TestFlatSliceArray(t *testing.T) {
	type st struct {
		Sf []float64 `SliceMaxSize:"5"`
		I  int
		Si []int
		Au [5]uint16
	}
	var s1, s2 st
	s1.I = 15
	s1.Si = []int{1, 2, 3}
	s1.Sf = []float64{1.1, 2.32, 0.55589}
	//s1.Au = [5]uint16{11, 12, 13, 14, 15}
	s1.Au[0] = 11
	s1.Au[1] = 12
	s1.Au[4] = 14
	//fmt.Println("s1:", s1)

	//fmt.Printf("s1: %p, si: %p, s2: %p, si: %p\n", &s1, &s1.Si, &s2, &s2.Si)

	container := parse(s1)
	//fmt.Println(container)
	b, _ := container.marshal(&s1, make([]byte, 0, 2048))
	//fmt.Printf("s1: %p, si: %p, s2: %p, si: %p\n", &s1, &s1.Si, &s2, &s2.Si)
	//fmt.Println("row:", container.Row(0))
	//fmt.Printf("s2-0: %d %d\n", len(s2.Si), cap(s2.Si))
	container.unmarshal(b, &s2)

	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestFlatSliceArray failed")
	}
	//fmt.Printf("s1: %p, si: %p, s2: %p, si: %p\n", &s1, &s1.Si, &s2, &s2.Si)
	//fmt.Println("s2:", s2)
	//fmt.Printf("s2-1: %d %d\n", len(s2.Si), cap(s2.Si))

	//fmt.Printf("s1.Si[0]: %p, %p\n", &s1.Si, &s1.Si[0])

	//var p *[]byte

	//p := (*[]byte)(unsafe.Pointer(&s1.Si))

	//fmt.Printf("s1.Si[0]: %p, %p\n", p, &(*p)[0])

}

func TestStringSliceArray(t *testing.T) {

	type st1 struct {
		Fs  []float32
		Str string
	}
	type st struct {
		I    int16
		SStr []string
		Ia   [5]uint8
		St1  st1
		Astr [2]string
	}

	var s1, s2 st
	s1.I = 12
	s1.SStr = append(s1.SStr, "hello")
	s1.SStr = append(s1.SStr, "word")
	s1.Ia = [5]uint8{11, 12}
	s1.Ia[3] = 23
	s1.St1.Str = "hello string"
	s1.St1.Fs = append(s1.St1.Fs, 1.23)
	s1.St1.Fs = append(s1.St1.Fs, 99.99)

	s1.Astr[0] = "good"
	s1.Astr[1] = "string array"

	//fmt.Println("s1:", s1)
	c := parse(s1)

	buf, _ := c.marshal(&s1, make([]byte, 0, 2048))
	//fmt.Println("rows: ", c.Row(0))

	c.unmarshal(buf, &s2)

	if !reflect.DeepEqual(s1, s2) {
		t.Error("Test string slice array failed")
	}
}

/*
func TestStructSliceArray(t *testing.T) {

		type st3 struct {
			I int
			S []string
			A [2]string
		}
		type st struct {
			I   int
			st3 []st3
			S   string
		}

		type ost struct {
			i   int
			Sst []st
			Ast [3]st
			str string
		}

		var v1, v2 ost

		v1.i = 10
		v1.str = "struct_slice"
		//st1 := st{ 12, "str"}

		var s3 st3
		s3.I = 41
		s3.S = append(s3.S, "hello s31")
		s3.A[0] = "hello s3 array 1"
		s3.A[0] = "hello s3 array 2"

		var st1 st
		st1.I = 12
		st1.S = "str"

		st1.st3 = append(st1.st3, s3)

		s3.S = append(s3.S, "hello s32")
		st1.st3 = append(st1.st3, s3)

		v1.Sst = append(v1.Sst, st1)
		st1.I = 14
		st1.S = "test"
		v1.Sst = append(v1.Sst, st1)

		v1.Ast[0].I = 21
		v1.Ast[0].S = "inner 1"

		v1.Ast[2].I = 31
		v1.Ast[2].S = "inner 3"

		v1.Ast[1].I = 31
		v1.Ast[1].S = "inner 3"

		t.Log("v1:", v1)
		//fmt.Printf("addr: %p %p\n", &v1.Sst[0], &v1.Sst[1])

		//fmt.Printf("addr a : %p %p %p\n", &v1.Ast[0], &v1.Ast[1], &v1.Ast[2])
		c := Parse(v1)
		c.Save(&v1)

		c.DumpRow(0, &v2)
		t.Log("v2:", v2)
		if !reflect.DeepEqual(v1, v2) {
			t.Error("Test struct slice array failed")
		}
	}
*/

func TestParse(t *testing.T) {
	//var i32 int32 = 0x7fffffff
	//fmt.Println("TestParse:", string(BOJECT_MAGIC), *(*int32)(Ptr(&BOJECT_MAGIC[0])), i32)

	s := make([]int, 10)
	s[5] = 5
	if len(s) != 10 {
		t.Error("TestMakeSlice len fail")
	}
	var s2 []int

	f := parse(s)
	b, _ := f.marshal(&s, make([]byte, 0, 2048))

	f.unmarshal(b, &s2)

	if !reflect.DeepEqual(s, s2) {
		t.Error("TestDirectSave failed")
	}
}

type stx struct {
	I uint16
	S string
}

func testBody[T comparable](in T, t *testing.T) {
	ti := parse(in)
	dest, _ := ti.marshal(Ptr(&in), make([]byte, 0, 2048))
	fmt.Println("TestTssdInt buf:", dest)

	//ti.print(dest)

	var out T
	ti.unmarshal(dest, Ptr(&out))
	fmt.Println("unmarshal in, out:", in, out)
	if in != out {
		t.Error("unmarshal failed")
	}
}

func equalSlice[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func testArray[T comparable](in []T, t *testing.T) {
	ti := parse(in)
	dest, _ := ti.marshal(Ptr(&in), make([]byte, 0, 2048))
	fmt.Println("testMergeArray buf:", dest)

	var out []T
	ti.unmarshal(dest, Ptr(&out))
	fmt.Println("unmarshal in, out:", in, out)
	if !equalSlice(in, out) {
		t.Error("unmarshal failed")
	}
}

func testBasicAndArray[T comparable](in []T, t *testing.T) {
	for i := range in {
		testBody(i, t)
	}
	testArray(in, t)
}

func testBasicAll[T comparable](in []T, t *testing.T) {
	testBasicAndArray(in, t)
	testBasicInStruct(in, t)
	testBasicInMap(in, in, t)

	inAll := make([]AllBasicType, len(in))
	for i := 0; i < len(in); i++ {
		(&inAll[i]).rand()
	}
	testBasicInMap(in, inAll, t)

	fmt.Println("testBasicAll: compost")

	testBasicInMap(in, makeCompostArray(in), t)
	testBasicInMap(in, makeCompost2Array(in), t)

}

func TestTssdAll(t *testing.T) {
	testBasicAll([]bool{true, false}, t)
	testBasicAll([]int8{0, -1, 1, 127, -128, 100, -35}, t)
	testBasicAll([]uint8{0, 1, 127, 255, 100}, t)
	testBasicAll([]uint16{0, 1, 127, 255, 12345, 0xFFFF}, t)
	testBasicAll([]int16{0, -1, 1, 127, -55, 255, -0x7FFF, 13579, 0x7FFF}, t)

	testBasicAll([]uint32{0, 1, 127, 255, 12345, 0xFFFF, 0xFFFFFFFF}, t)
	testBasicAll([]int32{0, -1, 1, 127, -55, 255, -0x7FFF, 13579, 0x7FFF, 0x7FFFFFFF, -0x7FFFFFFF}, t)

	testBasicAll([]uint64{0, 1, 127, 255, 12345, 0xFFFF, 0xFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, t)
	testBasicAll([]int64{0, -1, 1, 127, -55, 255, -0x7FFF, 13579, 0x7FFF, 0x7FFFFFFF,
		-0x7FFFFFFF, 0x7FFFFFFFFFFFFFFF, -0x7FFFFFFFFFFFFFFF}, t)
	testBasicAll([]uint{0, 1, 127, 255, 12345, 0xFFFF, 0xFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, t)
	testBasicAll([]int{0, -1, 1, 127, -55, 255, -0x7FFF, 13579, 0x7FFF, 0x7FFFFFFF,
		-0x7FFFFFFF, 0x7FFFFFFFFFFFFFFF, -0x7FFFFFFFFFFFFFFF}, t)

	testBasicAll([]float32{0.0, -1.23, 134.5, 12345.7890, -12898.0000}, t)
	testBasicAll([]float64{0.0, -9.23, 134.5, 123456789.7890, -12898786544444444.0000}, t)
	testBasicAll([]string{
		"",
		"a",
		" ",
		"           ",
		"",
		"a1",
		"aA",
		"5",
		"6677888888",
		"fooobar",
		"foo     bar",
		"password1234&*&***&* ###$$$afwewe",
	}, t)
}

func testBasicInStruct[T comparable](in []T, t *testing.T) {
	type st[T comparable] struct {
		Value T
		Slice []T
	}

	ti := parse(st[T]{})
	fn := func(stin *st[T]) {
		dest, _ := ti.marshal(Ptr(stin), make([]byte, 0, 2048))
		var out st[T]
		ti.unmarshal(dest, Ptr(&out))
		if stin.Value != out.Value || !equalSlice(stin.Slice, out.Slice) {
			t.Error("unmarshal failed")
		}
	}
	for i := range in {
		fn(&st[T]{Value: in[i]})
	}
	fn(&st[T]{Slice: in})
}

func testBasicInMap[T comparable, V any](in []T, in2 []V, t *testing.T) {
	var m = make(map[T]V, 0)

	ti := parse(m)
	//fmt.Println("testBasicInMap:", in[0], in[1])
	for i := range in {
		m[in[i]] = in2[i]
		dest, err := ti.marshal(Ptr(&m), make([]byte, 0, 2048))
		if err != nil {
			t.Error("testBasicInMap marshal failed")
		}
		//fmt.Println("testBasicInMap in[i]:", in[i], in2[i], len(dest))

		var out map[T]V
		_, err = ti.unmarshal(dest, Ptr(&out))
		if err != nil || !reflect.DeepEqual(m, out) {
			t.Error("testBasicInMap failed")
		}
	}
}

type AllBasicType struct {
	//we random the order
	Vuint32  uint32
	Vfloat64 float64
	Vuint8   uint8
	Vstring  string
	Vuint16  uint16
	Vint32   int32
	Vint64   int64
	Vbool    bool
	Vint16   int16
	Vuint64  uint64
	Vfloat32 float32
	Vint8    int8
}

type compost[T comparable] struct {
	AllBasicType
	M map[T]AllBasicType
	S []AllBasicType
}

func makeCompostArray[T comparable](in []T) []compost[T] {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r3 := r.Intn(3)
	fmt.Println("makeCompostArray r3:", r3)
	ret := make([]compost[T], len(in))

	for i := 0; i < len(in); i++ {
		(&ret[i].AllBasicType).rand()
		ret[i].M = make(map[T]AllBasicType, 0)
		for j := 0; j < r3; j++ {
			var a AllBasicType
			(&a).rand()
			ret[i].M[in[j]] = a
			ret[i].S = append(ret[i].S, a)
		}
	}
	return ret
}

type compost2[T comparable] struct {
	M []map[T][]AllBasicType
	//AllBasicType
}

func makeCompost2Array[T comparable](in []T) []compost2[T] {
	ret := make([]compost2[T], len(in))

	for i := 0; i < len(in); i++ {
		//(&ret[i].AllBasicType).rand()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r3 := r.Intn(3) + 1
		fmt.Println("makeCompostArray r3:", r3)
		mvalue := make([]AllBasicType, r3)
		for j := 0; j < r3; j++ {
			(&mvalue[j]).rand()
		}

		r3 = r.Intn(2) + 1

		ret[i].M = make([]map[T][]AllBasicType, r3)

		for j := 0; j < r3; j++ {
			ret[i].M[j] = make(map[T][]AllBasicType, 0)
			ret[i].M[j][in[j]] = mvalue
		}
	}
	return ret
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
	minUint32 = uint32(0)
	maxUint32 = ^uint32(0)

	minUint64 = uint64(0)
	maxUint64 = ^uint64(0)

	minInt32 = int32(-maxInt32 - 1)
	maxInt32 = int32(maxUint32 >> 1)

	minInt64 = int64(-maxInt64 - 1)
	maxInt64 = int64(maxUint64 >> 1)
)

func (this *AllBasicType) rand() {

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	this.Vbool = []bool{true, false}[r.Intn(10)%2]
	this.Vint8 = int8(r.Intn(255) - 128)
	this.Vuint8 = uint8(r.Intn(255))
	this.Vint16 = int16(r.Intn(0xFFFF) - (0xFFFF/2 + 1))
	this.Vuint16 = uint16(r.Intn(0xFFFF))
	this.Vint32 = int32(r.Intn(0xFFFFFFFF) - (0xFFFFFFFF/2 + 1))
	this.Vuint32 = uint32(r.Intn(0xFFFFFFFF))

	this.Vint64 = r.Int63() - int64(maxUint64/2)
	this.Vuint64 = uint64(r.Int63() * 2)
	this.Vstring = string(randBytes(r.Intn(255)))
	this.Vfloat32 = r.Float32()
	this.Vfloat64 = r.Float64()
}

func TestAllBasicTypeInStruct(t *testing.T) {
	var in, out AllBasicType
	ti := parse(in)

	(&in).rand()

	dest, _ := ti.marshal(Ptr(&in), make([]byte, 0, 2048))
	//fmt.Println("testAllBasicTypeInStruct buf:", dest)

	ti.unmarshal(dest, Ptr(&out))
	//fmt.Println("testAllBasicTypeInStruct unmarshal in, out:", in, out)
	if !reflect.DeepEqual(in, out) {
		t.Error("testAllBasicTypeInStruct unmarshal failed")
	}
}

func TestAllBasicTypeInStructArray(t *testing.T) {
	var in, out [3]AllBasicType
	ti := parse(in)

	for i := 0; i < 3; i++ {
		(&in[i]).rand()
	}

	dest, _ := ti.marshal(Ptr(&in[0]), make([]byte, 0, 2048))
	//fmt.Println("testAllBasicTypeInStruct buf:", dest)

	ti.unmarshal(dest, Ptr(&out))
	//fmt.Println("testAllBasicTypeInStruct unmarshal in, out:", in, out)
	if !reflect.DeepEqual(in, out) {
		t.Error("testAllBasicTypeInStruct unmarshal failed")
	}
}

func TestAllBasicTypeInStructSlice(t *testing.T) {
	var in, out []AllBasicType
	ti := parse(in)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := r.Intn(128)

	in = make([]AllBasicType, n)

	for i := 0; i < n; i++ {
		(&in[i]).rand()
	}

	fmt.Println("TestAllBasicTypeInStructSlice len:", len(in), " sizeof:", unsafe.Sizeof(in[0]))

	dest, _ := ti.marshal(Ptr(&in), make([]byte, 0, 2048))
	//fmt.Println("testAllBasicTypeInStruct buf:", dest)

	ti.unmarshal(dest, Ptr(&out))
	//fmt.Println("testAllBasicTypeInStruct unmarshal in, out:", in, out)
	if !reflect.DeepEqual(in, out) {
		t.Error("TestAllBasicTypeInStructSlice unmarshal failed")
	}
}

func TestTssdArray(t *testing.T) {
	var array = [4]int16{1, 2, 3, 4}
	ti := parse(array)

	dest, _ := ti.marshal(Ptr(&array), make([]byte, 0, 2048))

	fmt.Println("TestTssdArray2 buf:", dest)

	var j [4]int16
	_, err := ti.unmarshal(dest, Ptr(&j))

	fmt.Println("unmarshal array j:", array, j, err, cap(j))

	if err != nil || !reflect.DeepEqual(array, j) {
		t.Error("unmarsha array failed")
	}
}

func TestTssdSlice(t *testing.T) {
	var array = []int64{
		1, 2, 3, 4,
	}
	ti := parse(array)

	dest, _ := ti.marshal(Ptr(&array), make([]byte, 0, 2048))
	ti.print(dest)

	var j []int64
	_, err := ti.unmarshal(dest, Ptr(&j))
	fmt.Println("unmarshal slice j:", array, j, err)
	if err != nil || !reflect.DeepEqual(array, j) {
		t.Error("unmarsha slice failed")
	}
}

func TestTssdStringSlice(t *testing.T) {
	var array []string
	ti := parse(array)
	array = append(array, "hello")
	array = append(array, "world")

	dest, _ := ti.marshal(Ptr(&array), make([]byte, 0, 2048))
	ti.print(dest)

	var j []string
	_, err := ti.unmarshal(dest, Ptr(&j))
	fmt.Println("unmarshal TestTssdStringSlice j:", array, j, err)
	if err != nil || !reflect.DeepEqual(array, j) {
		t.Error("unmarsha TestTssdStringSlice failed")
	}
}

func TestTssdMap(t *testing.T) {
	var mp = map[string]int32{
		"12": 0x1234,
		"34": 0x5678,
	}
	ti := parse(mp)

	dest, _ := ti.marshal(Ptr(&mp), make([]byte, 0, 2048))

	fmt.Println("TestTssdMap buf2:", dest)
	ti.print(dest)

	var j map[string]int32

	_, err := ti.unmarshal(dest, Ptr(&j))
	fmt.Println("unmarshal map j:", mp, j, err)
	if err != nil || !reflect.DeepEqual(mp, j) {
		t.Error("unmarsha map failed")
	}
}

func TestTssdMapStructSlice(t *testing.T) {
	var mp []map[string]stx

	ti := parse(mp)

	mp = append(mp, map[string]stx{
		"12":  {345, "hello"},
		"foo": {6789, "bar"},
	})

	mp = append(mp, map[string]stx{
		"1278":    {45, "helllllo"},
		"foooooo": {789, "barrr"},
	})

	dest, _ := ti.marshal(Ptr(&mp), make([]byte, 0, 2048))

	fmt.Println("TestTssdMapStruct buf2:", dest)

	var j []map[string]stx

	_, err := ti.unmarshal(dest, Ptr(&j))
	fmt.Println("unmarshal TestTssdMapStruct j:", mp, j, err)
	if err != nil || !reflect.DeepEqual(mp, j) {
		t.Error("unmarsha TestTssdMapStruct failed")
	}
}

func TestTssdMapSliceValue(t *testing.T) {
	var mp = map[string][]string{
		"12":  {"345", "hello"},
		"foo": {"6789", "bar"},
	}

	ti := parse(mp)

	dest, _ := ti.marshal(Ptr(&mp), make([]byte, 0, 2048))

	fmt.Println("TestTssdMapStruct buf2:", dest)

	var j map[string][]string

	_, err := ti.unmarshal(dest, Ptr(&j))
	fmt.Println("unmarshal TestTssdMapStruct j:", mp, j, err)
	if err != nil || !reflect.DeepEqual(mp, j) {
		t.Error("unmarsha TestTssdMapStruct failed")
	}
}

func TestTssdPrint(t *testing.T) {

	s := stx{
		1234,
		"hello",
	}
	ti := parse(stx{})

	dest, _ := ti.marshal(Ptr(&s), make([]byte, 0, 2048))

	fmt.Println("===============TestTssdPrint===========================")
	ti.print(dest)

	var j stx
	_, err := ti.unmarshal(dest, Ptr(&j))
	if err != nil || j != s {
		t.Error("unmarshal struct failed:", s, j, err)
	}
	//fmt.Println("unmarshal struct s, j:", s, j)

	if !reflect.DeepEqual(s, j) {
		t.Error("unmarsha struct failed")
	}
}

func TestBuffer(t *testing.T) {
	buf := &Buffer{}
	buf.Append(nil)
	if buf.Size != 0 || len(buf.Data) != 1 || cap(buf.Data[0]) != TSSD_BUFFER_CAP {
		t.Error("Buffer Append nil err")
	}

	buf.Append([]byte(MAGIC))

	if !isMagic(buf.Data[0]) || buf.Size != len(MAGIC) {
		t.Error("Buffer Append MAGIC err")
	}
}

func TestBuffer2(t *testing.T) {
	buf := &Buffer{
		Cap: 2,
	}
	buf.Append(nil)
	if buf.Size != 0 || len(buf.Data) != 1 || cap(buf.Data[0]) != buf.Cap {
		t.Error("Buffer Append nil err")
	}

	buf.Append([]byte(MAGIC))
	if buf.Size != len(MAGIC) || len(buf.Data) != 3 || cap(buf.Data[0]) != buf.Cap {
		t.Error("Buffer Append magic err")
	}

	if string(buf.Data[0]) != string([]byte(MAGIC)[:buf.Cap]) ||
		string(buf.Data[1]) != string([]byte(MAGIC)[buf.Cap:buf.Cap*2]) ||
		string(buf.Data[2]) != string([]byte(MAGIC)[buf.Cap*2:]) {
		t.Error("Buffer Append magic content err")
	}

	if d, err := buf.Read(nil); err != nil || len(d) != 0 {
		t.Error("Buffer read 0 should return ok")
	}

	dest := make([]byte, 10)
	if _, err := buf.Read(dest[:0]); err != nil {
		t.Error("Buffer read empty should return ok")
	}

	if _, err := buf.Read(dest); err == nil {
		t.Error("Buffer read oversize should return err")
	}

	d, err := buf.Read(dest[:1])
	if err != nil || len(d) != 1 || d[0] != MAGIC[0] {
		t.Error("Buffer read 1 byte err:", err, d)
	}

	d, err = buf.Read(dest[:4])
	if err != nil || len(d) != 4 || string(d) != string(MAGIC[1:]) || buf.Size != 0 || buf.index != 2 || buf.pos != 1 {
		t.Error("Buffer read 4 bytes err:", err, d, buf)
	}

	if d, err = buf.Read(dest[:1]); err == nil {
		t.Error("Buffer read oversize should return err")
	}
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

func appendBuffer3(t *testing.T, first, second int, r1, r2 [][]byte) {
	buf := &Buffer{
		Cap: 3,
	}

	dest := make([]byte, 11)
	for i := range dest {
		dest[i] = byte(100 + i)
	}

	buf.Append(dest[:first])
	if !SliceSliceEqual(buf.Data, r1) {
		t.Error("Buffer append r1 err:", buf.Data, r1)
	}

	buf.Append(dest[:second])
	if !SliceSliceEqual(buf.Data, r2) {
		t.Error("Buffer append r2 err:", buf.Data, r2)
	}
}

func TestAppendBuffer3(t *testing.T) {
	appendBuffer3(t, 0, 0, [][]byte {[]byte{},},   [][]byte {[]byte{},})
	appendBuffer3(t, 0, 1, [][]byte {[]byte{},},   [][]byte {[]byte{100},})
	appendBuffer3(t, 1, 0, [][]byte {[]byte{100},},   [][]byte {[]byte{100},})
	appendBuffer3(t, 1, 1, [][]byte {[]byte{100},},   [][]byte {[]byte{100, 100},})
	appendBuffer3(t, 1, 2, [][]byte {[]byte{100},},   [][]byte {[]byte{100, 100, 101},})
	appendBuffer3(t, 1, 3, [][]byte {[]byte{100},},   [][]byte {[]byte{100, 100, 101},[]byte{102}})
	appendBuffer3(t, 1, 4, [][]byte {[]byte{100},},   [][]byte {[]byte{100, 100, 101},[]byte{102, 103}})
	appendBuffer3(t, 1, 5, [][]byte {[]byte{100},},   [][]byte {[]byte{100, 100, 101},[]byte{102, 103, 104}})

	appendBuffer3(t, 2, 1, [][]byte {[]byte{100, 101},},   [][]byte {[]byte{100, 101, 100},})
	appendBuffer3(t, 2, 2, [][]byte {[]byte{100, 101},},   [][]byte {[]byte{100, 101, 100}, []byte{101}})
	appendBuffer3(t, 2, 3, [][]byte {[]byte{100, 101},},   [][]byte {[]byte{100, 101, 100}, []byte{101, 102}})
	appendBuffer3(t, 2, 4, [][]byte {[]byte{100, 101},},   [][]byte {[]byte{100, 101, 100}, []byte{101, 102, 103}})
	appendBuffer3(t, 2, 5, [][]byte {[]byte{100, 101},},   [][]byte {[]byte{100, 101, 100}, []byte{101, 102, 103}, []byte{104}})
}


func readBuffer3(t *testing.T, first, second int, r1, r2 []byte) {
	buf := &Buffer{
		Cap: 3,
	}

	dest := make([]byte, 11)
	for i := range dest {
		dest[i] = byte(100 + i)
	}

	buf.Append(dest)

	for i := range dest {
		dest[i] = byte(0)
	}
	d, err := buf.Read(dest[:first])
	if err != nil || !SliceEqual(d, r1) || !SliceEqual(dest[:first], r1){
		t.Error("Buffer read r1 err:", err, d, r1)
	}

	for i := range dest {
		dest[i] = byte(0)
	}
	d, err = buf.Read(dest[:second])
	if err != nil || !SliceEqual(d, r2) || !SliceEqual(dest[:second], r2){
		t.Error("Buffer read 4 bytes err:", err, d, r2)
	}
}

func TestReadBuffer3(t *testing.T) {
	readBuffer3(t, 0, 1, []byte{}, []byte{100})
	readBuffer3(t, 1, 0, []byte{100}, []byte{})
	readBuffer3(t, 1, 1, []byte{100}, []byte{101})
	readBuffer3(t, 1, 2, []byte{100}, []byte{101, 102})
	readBuffer3(t, 1, 3, []byte{100}, []byte{101, 102, 103})
	readBuffer3(t, 1, 4, []byte{100}, []byte{101, 102, 103, 104})
	readBuffer3(t, 1, 5, []byte{100}, []byte{101, 102, 103, 104, 105})
	readBuffer3(t, 1, 6, []byte{100}, []byte{101, 102, 103, 104, 105, 106})
	readBuffer3(t, 1, 7, []byte{100}, []byte{101, 102, 103, 104, 105, 106, 107})
	readBuffer3(t, 2, 1, []byte{100, 101}, []byte{102})
	readBuffer3(t, 2, 2, []byte{100, 101}, []byte{102, 103})
	readBuffer3(t, 2, 3, []byte{100, 101}, []byte{102, 103, 104})
	readBuffer3(t, 2, 4, []byte{100, 101}, []byte{102, 103, 104, 105})
	readBuffer3(t, 2, 5, []byte{100, 101}, []byte{102, 103, 104, 105, 106})
	readBuffer3(t, 2, 6, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107})
	readBuffer3(t, 2, 7, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107, 108})
	readBuffer3(t, 2, 8, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107, 108, 109})
	readBuffer3(t, 2, 9, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107, 108, 109, 110})
}
