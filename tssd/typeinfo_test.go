package tssd

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"
	//"strconv"
	//"assert"
	//tssd "github.com/simpleKV/tssd/tssd"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var ignoreTagOpt = cmp.FilterPath(func(p cmp.Path) bool {
	return strings.HasSuffix(p.GoString(), "Ignore")
}, cmp.Ignore())

// in & out should be address both
func doMarshalUnmarshal(t *testing.T, in any, out any, opts ...cmp.Option) {

	//ti := parse(in)
	value := reflect.ValueOf(in)
	v := value.Type().Elem()
	ti := parse(reflect.New(v).Elem().Interface())

	dest, err := ti.marshal(value.UnsafePointer())
	if err != nil {
		t.Error("doMarshalUnmarshal marshal fail:", err)
	}
	if err = ti.unmarshal(dest, reflect.ValueOf(out).UnsafePointer()); err != nil {
		t.Error("doMarshalUnmarshal unmarshal fail:", err)
	}

	if !cmp.Equal(in, out, opts...) {
		t.Error("cmp equal fail")
	}
}

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

	buff := &Buffer{}
	e := container.marshalTo(&sin1, buff)
	if e != nil || len(buff.fragments[0].payload) == 0 {
		t.Errorf("Test String Marshal err %s", e)
	}

	fmt.Println("testString out buf:", buff)
	container.print(*buff)
	fmt.Println("testString out end")

	container.unmarshal(buff, &s2)
	if !reflect.DeepEqual(sin1, s2) {
		t.Errorf("Test String err: [%s], [%s]", sin1.Str, s2.Str)
	}

	buff2 := &Buffer{}
	e = container.marshalTo(&sin2, buff2)
	if e != nil {
		t.Errorf("Test String Marshal err %s", e)
	}
	container.print(*buff2)
	container.unmarshal(buff2, &s2)
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
	doMarshalUnmarshal(t, &sin, &sout)
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
	doMarshalUnmarshal(t, &sin1, &s2)

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
	doMarshalUnmarshal(t, &sin1, &s2)
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
	doMarshalUnmarshal(t, &in1, &s2)
}

func TestSliceUint(t *testing.T) {
	in1 := []int8{1, 2, 3, 5, 4}
	var s2 []int8
	doMarshalUnmarshal(t, &in1, &s2)
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

	doMarshalUnmarshal(t, &in1, &s2)

	in1 = ss{
		[]int{},
		[]string{},
		[]int64{},
		[]byte{},
	}
	doMarshalUnmarshal(t, &in1, &s2)
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
	doMarshalUnmarshal(t, &in1, &s2)
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
	doMarshalUnmarshal(t, &in1, &s2)
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
	doMarshalUnmarshal(t, &in1, &s2)
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
	n, _ := container.marshal(&in1)

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
		I  uint32
		Ts []time.Time
		//str string
		Ss []string
	}

	type sout struct {
		I    []int16
		Nest [4]sin
		Strs []string
	}
	now := time.Now()
	in1 := sout{
		[]int16{10, 123},
		//[2]sin{{1, "hello"}, {2, ""}},
		[4]sin{{123, []time.Time{now}, []string{""}}, {2, []time.Time{now, now}, []string{"", "abc"}}, {0, []time.Time{now}, []string{"", ""}}, {234, []time.Time{now}, []string{"abc", ""}}},
		[]string{"abc", "", "abcd", "a"},
	}
	fmt.Println("Test TestNestStructArray begin ~~~~~~~~~~~~~~")
	var s2 sout
	container := parse(in1)

	is := []int8{Tobject, 3, 0, Tarraym, Tint16, Tarray, Tobject, 3, 0, Tuint32, Tarray, Ttime, Tstring, Tarray, Tstring, Tarray, Tstring}

	if !TypesEqual(container.types(), is) {
		t.Errorf("Test TestNestStructArray types fail")
	}

	n, _ := container.marshal(&in1)
	fmt.Println("in1:", in1)

	container.unmarshal(n, &s2)
	//we cmp time first
	for i := 0; i < len(in1.Nest); i++ {
		if !TimeSliceEqual(in1.Nest[i].Ts, s2.Nest[i].Ts) {
			t.Errorf("Test TestNestStructArray time err")
		}
		in1.Nest[i].Ts = nil
		s2.Nest[i].Ts = nil
	}

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

	doMarshalUnmarshal(t, &s1, &s2)
}

func TestSimpleTime(t *testing.T) {

	tt := time.Now()
	container := parse(tt)
	b, _ := container.marshal(&tt)

	var v2 time.Time
	container.unmarshal(b, &v2)
	fmt.Println("tt:", tt.Format(time.RFC3339Nano))
	fmt.Println("v2:", v2.Format(time.RFC3339Nano))
	if !v2.Equal(tt) {
		t.Error("TestSimpleTime failed")
	}
}

func TestSimpleTimeArray(t *testing.T) {

	type st struct {
		Tt []time.Time
	}

	tt := time.Now()
	s1 := st{
		[]time.Time{tt, tt.AddDate(-22, 1, 2)},
	}
	var s2 st

	doMarshalUnmarshal(t, &s1, &s2)
}

func TestEmbedStruct(t *testing.T) {
	v := TestStruct{V: 2, T: time.Now()}
	v.S1.V = 3
	var v2 TestStruct
	doMarshalUnmarshal(t, &v, &v2)
}

func TestTime(t *testing.T) {
	v := TestStruct{V: 2, T: time.Now()}
	v.S1.V = 3
	var v2 TestStruct
	doMarshalUnmarshal(t, &v, &v2)
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
	doMarshalUnmarshal(t, &s1, &s2)
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
	doMarshalUnmarshal(t, &s1, &s2)
}

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
	doMarshalUnmarshal(t, &v1, &v2, cmpopts.IgnoreUnexported(ost{}), cmpopts.IgnoreUnexported(st{}), cmpopts.IgnoreUnexported(st3{}))
}

func TestParse(t *testing.T) {
	//var i32 int32 = 0x7fffffff
	//fmt.Println("TestParse:", string(BOJECT_MAGIC), *(*int32)(Ptr(&BOJECT_MAGIC[0])), i32)
	s := make([]int, 10)
	s[5] = 5
	if len(s) != 10 {
		t.Error("TestMakeSlice len fail")
	}
	var s2 []int
	doMarshalUnmarshal(t, &s, &s2)
}

func testBody[T comparable](in T, t *testing.T) {
	ti := parse(in)
	dest, err := ti.marshal(Ptr(&in))
	if err != nil {
		t.Error("testBody marshal err:", err)
	}
	//ti.print(dest)
	var out T
	err = ti.unmarshal(dest, Ptr(&out))
	if err != nil {
		t.Error("testBody unmarshal err:", err)
	}
}

func testArray[T comparable](in []T, t *testing.T) {
	ti := parse(in)
	dest, _ := ti.marshal(Ptr(&in))
	fmt.Println("testMergeArray buf:", dest)

	var out []T
	ti.unmarshal(dest, Ptr(&out))
	fmt.Println("unmarshal in, out:", in, out)
	if !SliceEqual(in, out) {
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
		dest, _ := ti.marshal(Ptr(stin))
		var out st[T]
		ti.unmarshal(dest, Ptr(&out))
		if stin.Value != out.Value || !SliceEqual(stin.Slice, out.Slice) {
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
		dest, err := ti.marshal(Ptr(&m))
		if err != nil {
			t.Error("testBasicInMap marshal failed")
		}
		//fmt.Println("testBasicInMap in[i]:", in[i], in2[i], len(dest))

		var out map[T]V
		err = ti.unmarshal(dest, Ptr(&out))
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
	(&in).rand()
	doMarshalUnmarshal(t, &in, &out)
}

func TestAllBasicTypeInStructArray(t *testing.T) {
	var in, out [3]AllBasicType
	ti := parse(in)

	for i := 0; i < 3; i++ {
		(&in[i]).rand()
	}

	dest, _ := ti.marshal(Ptr(&in[0]))
	//fmt.Println("testAllBasicTypeInStruct buf:", dest)

	ti.unmarshal(dest, Ptr(&out))
	//fmt.Println("testAllBasicTypeInStruct unmarshal in, out:", in, out)
	if !reflect.DeepEqual(in, out) {
		t.Error("testAllBasicTypeInStruct unmarshal failed")
	}
	doMarshalUnmarshal(t, &in, &out)
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

	buf := &Buffer{}
	ti.marshalTo(Ptr(&in), buf)
	//fmt.Println("testAllBasicTypeInStruct buf:", dest)

	ti.unmarshal(buf, Ptr(&out))
	//fmt.Println("testAllBasicTypeInStruct unmarshal in, out:", in, out)
	if !reflect.DeepEqual(in, out) {
		t.Error("TestAllBasicTypeInStructSlice unmarshal failed")
	}
}

func TestTssdArray(t *testing.T) {
	var array = [4]int16{1, 2, 3, 4}
	var j [4]int16
	doMarshalUnmarshal(t, &array, &j)
}

func TestTssdSlice(t *testing.T) {
	var array = []int64{
		1, 2, 3, 4,
	}
	var j []int64
	doMarshalUnmarshal(t, &array, &j)
}

func TestTssdStringSlice(t *testing.T) {
	var array []string
	ti := parse(array)
	array = append(array, "hello")
	array = append(array, "world")

	dest, _ := ti.marshal(Ptr(&array))
	ti.print(*dest)

	var j []string
	err := ti.unmarshal(dest, Ptr(&j))
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
	var j map[string]int32
	doMarshalUnmarshal(t, &mp, &j)
}

type stx struct {
	I       uint16
	BIgnore string `tssd:"-,"` // tags ignore
	S       string
	a       int32 // unexported
}

func TestTssdMapStructSlice(t *testing.T) {
	var mp []map[string]stx

	ti := parse(mp)

	mp = append(mp, map[string]stx{
		"12":  {345, "2", "hello", 1},
		"foo": {6789, "3", "bar", 2},
	})

	mp = append(mp, map[string]stx{
		"1278":    {45, "4", "helllllo", 3},
		"foooooo": {789, "5", "barrr", 4},
	})

	dest, _ := ti.marshal(Ptr(&mp))

	fmt.Println("TestTssdMapStruct buf2:", dest)

	var j []map[string]stx

	err := ti.unmarshal(dest, Ptr(&j))
	fmt.Println("unmarshal TestTssdMapStruct j:", mp, j, err)

	fmt.Println("mp:", mp)
	fmt.Println("j:", j)
	fmt.Println("diff: ", cmp.Diff(mp, j, cmpopts.IgnoreUnexported(stx{}), ignoreTagOpt))

	if err != nil || !cmp.Equal(mp, j, cmpopts.IgnoreUnexported(stx{}), ignoreTagOpt) {
		t.Error("cmp equal fail")
	}
}

func TestTssdMapSliceValue(t *testing.T) {
	var mp = map[string][]string{
		"12":  {"345", "hello"},
		"foo": {"6789", "bar"},
	}
	var j map[string][]string

	doMarshalUnmarshal(t, &mp, &j)
}

func TestTssdPrint(t *testing.T) {
	s := stx{
		1234,
		"ssss",
		"hello",
		5,
	}
	var j stx
	doMarshalUnmarshal(t, &s, &j, cmpopts.IgnoreUnexported(stx{}), ignoreTagOpt)
}

func TestTssdTypes(t *testing.T) {
	type st[T comparable] struct {
		Value T
	}

	if !TypesEqual(parse(st[int8]{}).types(), []int8{Tobject, 1, 0, Tint8}) {
		t.Error("parse int8 types error")
	}
	if !TypesEqual(parse(st[uint64]{}).types(), []int8{Tobject, 1, 0, Tuint64}) {
		t.Error("parse uint64 types error")
	}
	if !TypesEqual(parse(st[float64]{}).types(), []int8{Tobject, 1, 0, Tfloat64}) {
		t.Error("parse st float64 types error")
	}
	type st2[T, T2 comparable] struct {
		Value  T
		Value2 T2
	}
	if !TypesEqual(parse(st2[uint8, int32]{}).types(), []int8{Tobject, 2, 0, Tuint8, Tint32}) {
		t.Error("parse st2 uint8/int32 types error")
	}
	if !TypesEqual(parse(st2[float32, bool]{}).types(), []int8{Tobject, 2, 0, Tfloat32, Tbool}) {
		t.Error("parse st2 float32/bool types error")
	}
	type st3 struct {
		Value  string
		Value2 []int16
		T      time.Time
	}
	if !TypesEqual(parse(st3{}).types(), []int8{Tobject, 3, 0, Tstring, Tarraym, Tint16, Ttime, Tstring}) {
		t.Error("parse st3 types error")
	}

	type st4 struct {
		Value2 []string
		T      []time.Time
		M      map[string]st3
	}
	if !TypesEqual(parse(st4{}).types(), []int8{Tobject, 3, 0, Tarray, Tstring, Tarray, Ttime, Tstring, Tdict, Tdictk, Tstring, Tdictv, Tobject, 3, 0, Tstring, Tarraym, Tint16, Ttime, Tstring}) {
		t.Error("parse st4 types error")
	}

}

type stIgnoreTestIn struct {
	Value   int16
	a       string //unexp
	BIgnore int32  `tssd:"xxx,-,yyy"`
	B       byte
}

type stIgnoreTest struct {
	AIgnore stIgnoreTestIn `tssd:"other,-,123"`
	S       stIgnoreTestIn
	unexp   int8
	Str     string
}

func TestUnexportedAndIgnoreFields(t *testing.T) {
	s1 := stIgnoreTestIn{
		123,
		"astring",
		456,
		32,
	}
	var s2 stIgnoreTestIn

	doMarshalUnmarshal(t, &s1, &s2, cmpopts.IgnoreUnexported(stIgnoreTestIn{}), ignoreTagOpt)
}

func TestUnexportedAndIgnoreFieldsNest(t *testing.T) {
	s1 := stIgnoreTestIn{
		123,
		"astring",
		456,
		32,
	}

	s2 := stIgnoreTestIn{
		1234,
		"xxxastring",
		456789,
		23,
	}
	s3 := stIgnoreTest{
		s1,
		s2,
		7,
		"Hello TSSD",
	}

	var s4 stIgnoreTest
	doMarshalUnmarshal(t, &s3, &s4, cmpopts.IgnoreUnexported(stIgnoreTest{}), cmpopts.IgnoreUnexported(stIgnoreTestIn{}), ignoreTagOpt)
}

func TestUnexportedAndIgnoreFieldsSlice(t *testing.T) {
	s1 := stIgnoreTestIn{
		123,
		"astring",
		456,
		32,
	}

	s2 := stIgnoreTestIn{
		1234,
		"xxxastring",
		456789,
		23,
	}
	s3 := stIgnoreTest{
		s1,
		s2,
		7,
		"Hello TSSD",
	}
	s4 := stIgnoreTest{
		s1,
		s2,
		88,
		"Hello world",
	}
	in := []stIgnoreTest{s3, s4}
	var out []stIgnoreTest
	doMarshalUnmarshal(t, &in, &out, cmpopts.IgnoreUnexported(stIgnoreTest{}), cmpopts.IgnoreUnexported(stIgnoreTestIn{}), ignoreTagOpt)
}

func TestUnexportedAndIgnoreFieldsMap(t *testing.T) {
	s1 := stIgnoreTestIn{
		123,
		"astring",
		456,
		32,
	}

	s2 := stIgnoreTestIn{
		1234,
		"xxxastring",
		456789,
		23,
	}
	s3 := stIgnoreTest{
		s1,
		s2,
		7,
		"Hello TSSD",
	}
	s4 := stIgnoreTest{
		s1,
		s2,
		88,
		"Hello world",
	}
	in := map[string]stIgnoreTest{
		"hello": s3,
		"foo":   s4,
	}
	var out map[string]stIgnoreTest
	doMarshalUnmarshal(t, &in, &out, cmpopts.IgnoreUnexported(stIgnoreTest{}), cmpopts.IgnoreUnexported(stIgnoreTestIn{}), ignoreTagOpt)
}
