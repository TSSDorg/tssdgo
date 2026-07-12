package tssd

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

type st[K comparable, V any] struct {
	i   int
	str string
	m   map[K]V
}

type stout struct {
	i int
	s st[string, int]
}

func (this *st[K, V]) GetMapPtr() *map[K]V {
	return &this.m
}

func (this *st[K, V]) MapPtr(p Ptr) *map[K]V {
	return (*map[K]V)(p)
}

func TestMap2(t *testing.T) {

	mi := st[string, int]{
		1,
		"hello",
		map[string]int{
			"hello": 123,
		},
	}

	v := reflect.ValueOf(&mi.m)

	fmt.Println("TestMap:", mi, reflect.TypeOf(mi))
	fmt.Printf("testmap: %p %x %p\n", v.UnsafePointer(), v.Pointer(), Ptr(&mi.m))

	m3 := Ptr(&mi.m)

	m := reflect.ValueOf(mi.MapPtr(m3)).Elem()
	fmt.Println("testMap_basics:", m.Kind(), m.Type().Key(), m.Type().Elem())
	if m.Kind() == reflect.Map {
		newInstance := reflect.MakeMap(m.Type())
		keys := m.MapKeys()
		for _, k := range keys {
			key := k.Convert(newInstance.Type().Key())
			value := m.MapIndex(key)
			newInstance.SetMapIndex(key, value)

			fmt.Println("======TestMap:", m.Kind(), &k, &key, value.CanInterface(), value.Interface(), value.CanAddr(), &value, value.Type().Kind())

		}
		fmt.Printf("newInstance = %v\n", newInstance)
	}

	m2 := reflect.ValueOf([]struct {
		i int
		s []string
	}{})
	fmt.Println("TestMap:", m2.Kind())

}

func TestMap_d2(t *testing.T) {
	var m1 map[string]interface{}

	p := Ptr(&m1)

	pv := reflect.NewAt(reflect.TypeOf(m1), p).Elem()
	pv.Set(reflect.MakeMap(reflect.TypeOf(m1)))
	fmt.Println("m1 is nil:", m1 == nil, m1, reflect.TypeOf(m1), reflect.TypeOf(m1).Key(), reflect.TypeOf(m1).Elem())

	pv.SetMapIndex(reflect.ValueOf("today"), reflect.ValueOf("is monday"))
	fmt.Println("m1 is nil:", m1 == nil, m1)

	var str string
	var byt []byte
	//var i int
	var b bool
	//v := reflect.ValueOf(i)
	fmt.Println("TestMap_d2: ", unsafe.Sizeof(str), unsafe.Sizeof(byt), unsafe.Sizeof(b), unsafe.Sizeof(byt[0]))

}

type stin struct {
	Str string
	I   int
	St  stin2
}

type stin2 struct {
	Str string
	I   int
}

type stmap struct {
	I int
	M map[string]stin
}

/*
func TestSaveDumpEmptyMapStruct(t *testing.T) {
	fmt.Println("TestSaveDumpMapStruct")

	s1 := stmap{}
	s1.I  = 1
	s1.M = nil

	c := parse(s1)
	buf, _ := c.marshal(&s1, make([]byte, 0, 2048))
	fmt.Println(c)

	var s2 stmap
	c.unmarshal(buf, &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapStruct error:", s1, s2)
	}
}
*/

func TestSaveDumpMapStruct(t *testing.T) {
	fmt.Println("TestSaveDumpMapStruct")

	s1 := stmap{
		1,
		map[string]stin{
			"hello": stin{
				"world",
				123,
				stin2{
					"abc",
					456,
				},
			},
		},
	}

	c := parse(s1)
	buf, _ := c.marshal(&s1)
	fmt.Println(c)
	fmt.Println("buf:", buf)

	var s2 stmap
	_, err := c.unmarshal(buf.Data[0], &s2)

	fmt.Println("s1:", s1, " err:", err, " s2:", s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapStruct error")
	}
}

func TestSaveDumpMapSimple(t *testing.T) {
	fmt.Println("TestSaveDumpMapSimple")

	type stmap struct {
		I   int
		M   map[string]int
		M0  map[string]uint
		M1  map[string]bool
		M2b map[string]byte
		M2  map[string]int8
		M3  map[string]uint8
		M4  map[string]int16
		M5  map[string]uint16
		M6  map[string]int32
		M7  map[string]uint32
		M8  map[string]int64
		M9  map[string]uint64
		M10 map[string]float32
		M11 map[string]float64
	}

	s1 := stmap{
		1,
		map[string]int{
			"kint": -123,
		},
		map[string]uint{
			"kint": 123,
		},
		map[string]bool{
			"kbool": true,
		},
		map[string]byte{
			"kint8": 8,
		},
		map[string]int8{
			"kint8": -8,
		},
		map[string]uint8{
			"kuint8": 8,
		},
		map[string]int16{
			"kint8": -16,
		},
		map[string]uint16{
			"kuint8": 16,
		},
		map[string]int32{
			"kint32": -32,
		},
		map[string]uint32{
			"kuint32": 32,
		},
		map[string]int64{
			"kint64": -64,
		},
		map[string]uint64{
			"kuint64": 64,
		},
		map[string]float32{
			"kfloat32": -32.0,
		},
		map[string]float64{
			"kfloat64": 64.0,
		},
	}

	c := parse(s1)
	buf, _ := c.marshal(&s1)
	fmt.Println(c)

	var s2 stmap
	c.unmarshal(buf.Data[0], &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapSimple error")
	}
}

func TestSaveDumpMapString(t *testing.T) {
	fmt.Println("TestSaveDumpMapString")

	type stmap struct {
		I int
		M map[int][]string
	}

	s1 := stmap{
		1,
		map[int][]string{
			123: {"helool", "wooold"},
		},
	}

	c := parse(s1)
	buf, _ := c.marshal(&s1)
	fmt.Println(c)

	var s2 stmap
	c.unmarshal(buf.Data[0], &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapString error")
	}
}

func TestSaveDumpMapSimpleSlice(t *testing.T) {
	

	type stmap struct {
		I int
		M map[string][]int
	}

	s1 := stmap{
		1,
		map[string][]int{
			"hello": []int{123, 456},
		},
	}

	c := parse(s1)
	buf, _ := c.marshal(&s1)

	var s2 stmap
	c.unmarshal(buf.Data[0], &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapSimpleSlice error")
	}
		
}

func TestSaveDumpMapStructSlice(t *testing.T) {
	fmt.Println("TestSaveDumpMapSimpleSlice")

	type stmap struct {
		I int
		M map[string][]stin2
	}

	s1 := stmap{
		1,
		map[string][]stin2{
			"hello": {
				{"h1", 111},
				{"h2", 222},
			},
		},
	}

	c := parse(s1)
	buf, _ := c.marshal(&s1)
	fmt.Println(c)

	var s2 stmap
	c.unmarshal(buf.Data[0], &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapSimpleSlice error")
	}
}

func TestSaveDumpMapStructSlice2(t *testing.T) {
	fmt.Println("TestSaveDumpMapSimpleSlice")

	type stmap struct {
		I int
		M map[string][]stin
	}

	var s0 stmap

	s1 := stmap{
		1,
		map[string][]stin{
			"hello": {
				{"h1", 111, stin2{"h3", 333}},
				{"h2", 222, stin2{"h4", 444}},
			},
		},
	}

	c := parse(s0)
	buf, _ := c.marshal(&s1)
	fmt.Println(c)

	var s2 stmap
	c.unmarshal(buf.Data[0], &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapSimpleSlice error")
	}
}

func TestSaveDumpMapSlice(t *testing.T) {
	fmt.Println("TestSaveDumpMapSlice")

	type stmap struct {
		I int
		M []map[string]stin2
	}

	s1 := stmap{
		1,
		[]map[string]stin2{
			{
				"hello": {"h1", 111},
			},
			{
				"hello2": {"h2", 222},
			},
		},
	}

	c := parse(s1)
	buf, _ := c.marshal(&s1)
	fmt.Println(c)

	var s2 stmap
	c.unmarshal(buf.Data[0], &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapSlice error")
	}
}

func TestSaveDumpMapArray(t *testing.T) {
	fmt.Println("TestSaveDumpMapArray")

	type stmap struct {
		I int
		M [2]map[string]stin2
	}

	s1 := stmap{
		1,
		[2]map[string]stin2{
			{
				"hello": {"h1", 111},
			},
			{
				"hello2": {"h2", 222},
			},
		},
	}

	c := parse(s1)
	buf, _ := c.marshal(&s1)
	fmt.Println(c)

	var s2 stmap
	c.unmarshal(buf.Data[0], &s2)

	fmt.Println(s1, s2)
	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestSaveDumpMapSlice error")
	}
}

func TestMapArraySlice(t *testing.T) {
	type si struct {
		S string
		M []map[string]string
	}

	type so struct {
		I  int
		Si si
		S  string
		M  [2]map[string]int
	}

	var si1 si
	si1.S = "hello"
	si1.M = append(si1.M, make(map[string]string))
	si1.M[0]["h1"] = "v1"

	si1.M = append(si1.M, make(map[string]string))

	si1.M[1]["h2"] = "v2"
	si1.M[1]["h3"] = "v3"

	var s1, s2 so

	s1.I = 123
	s1.Si = si1
	s1.S = "sfe"

	s1.M[0] = make(map[string]int)
	s1.M[0]["o1"] = 1332

	s1.M[1] = make(map[string]int)
	s1.M[1]["o2"] = 133

	container := parse(s1)
	buf, _ := container.marshal(&s1)

	container.unmarshal(buf.Data[0], &s2)

	fmt.Println("s1:", s1, ", s2:", s2)

	//s2.i = 13

	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestMap failed")
	}
}


func TestMapInMapEmpty(t *testing.T) {
	type si struct {
		S string
		M map[string]string
	}

	type so struct {
		I  int
		M  map[string]si
	}

	var si1 si
	si1.S = "hello"
	//si1.M = append(si1.M, make(map[string]string))
	si1.M = make(map[string]string)
	//si1.M[0]["h1"] = "v1"

	//si1.M = append(si1.M, make(map[string]string))

	//si1.M[1]["h2"] = "v2"
	//si1.M[1]["h3"] = "v3"

	var s1, s2 so
	container := parse(s1)

	s1.I = 123
	//s1.Si = si1
	//s1.S = "sfe"

	s1.M= make(map[string]si)
	s1.M["o1"] = si1

	//s1.M[1] = make(map[string]int)
	//s1.M[1]["o2"] = 133

	
	buf, _ := container.marshal(&s1)

	container.unmarshal(buf.Data[0], &s2)

	fmt.Println("s1:", s1, ", s2:", s2)

	//s2.i = 13

	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestMap failed")
	}
}


func TestMapInMap(t *testing.T) {
	type si struct {
		S string
		M map[string]string
	}

	type so struct {
		I  int
		M  map[string]si
	}

	var si1 si
	si1.S = "hello"
	//si1.M = append(si1.M, make(map[string]string))
	si1.M = make(map[string]string)
	si1.M["h1"] = "v1"

	//si1.M = append(si1.M, make(map[string]string))

	//si1.M[1]["h2"] = "v2"
	//si1.M[1]["h3"] = "v3"

	var s1, s2 so
	container := parse(s1)

	s1.I = 123
	//s1.Si = si1
	//s1.S = "sfe"

	s1.M= make(map[string]si)
	s1.M["o1"] = si1

	//s1.M[1] = make(map[string]int)
	//s1.M[1]["o2"] = 133

	
	buf, _ := container.marshal(&s1)

	container.unmarshal(buf.Data[0], &s2)

	fmt.Println("s1:", s1, ", s2:", s2)

	//s2.i = 13

	if !reflect.DeepEqual(s1, s2) {
		t.Error("TestMap failed")
	}
}
