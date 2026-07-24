package tssd_test

import (
	//"errors"
	//"crypto/md5"
	//"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	tssd "github.com/tssdorg/tssdgo/tssd"
)

//this file demo how to override Flatable.Schema()

func init() {
	/*
		tssd.HashFunc = func(data []byte) []byte {
			hasher := md5.New()
			hasher.Write(data)                         // Write the data to the hasher
			hashBytes := hasher.Sum(nil)                // Get the hash sum as a byte slice
			hashString := hex.EncodeToString(hashBytes) // Convert to a hex string
			l := len(hashString)
			return []byte(hashString[:4] + hashString[l-6:l])
		}

		tssd.ChecksumFunc = func(data []byte) []byte {
			hasher := md5.New()
			hasher.Write(data)                         // Write the data to the hasher
			hashBytes := hasher.Sum(nil)                // Get the hash sum as a byte slice
			hashString := hex.EncodeToString(hashBytes) // Convert to a hex string
			l := len(hashString)
			return []byte(hashString[:5] + hashString[l-8:l])
		}*/
}

const WORKER_GROUP_NAME = "work_group_name"

// you can alias to simplify for users,
// but update it after every update the struct
type worker = worker_V2

// //////////////////////////the worker V2////////////////////////////////////
// after you need add/update the struct
// you need implment Progeny() string to specify which version
// and rename a new class name
type worker_V2 struct {
	tssd.Flat[worker_V2, *worker_V2]
	Age     int16
	Address string //the new version of worker, which we add a new field Address
	Name    string
}

func (this *worker_V2) Group() string {
	return WORKER_GROUP_NAME
}

func (this *worker_V2) Version() string {
	return "worker_V2"
}

func (this *worker_V2) Decorate(flat tssd.Flatable) tssd.Flatable {
	old := flat.(*worker_V1)
	this.Name = old.Name
	this.Age = old.Age
	this.Address = defaultAddress
	return this
}

func (this *worker_V2) Progeny() string {
	return "worker_V3"
}

// generate an uniqu id
func (this *worker_V2) TID() string {
	return "worker_V2-" + strconv.Itoa(tid)
}

func (this *worker_V2) Schema() tssd.Schema {
	return tssd.Schema{
		-1,
		string(tssd.HashFunc(this.Types())),
		this.TID(),
		"you can put a json object string",
	}
}

// //////////////////////////the worker V1////////////////////////////////////
// after you need add/update the struct
// you need implment Progeny() string to specify which version
// and rename a new class name
type worker_V1 struct {
	tssd.Flat[worker_V1, *worker_V1]
	Name string
	Age  int16
}

func (this *worker_V1) Group() string {
	return WORKER_GROUP_NAME
}

func (this *worker_V1) Version() string {
	return "worker_V1"
}

func (this *worker_V1) Progeny() string {
	return "worker_V2"
}

var tid int

// generate an uniqu id
func (this *worker_V1) TID() string {
	tid++
	return strconv.Itoa(tid)
}

const (
	SCHEMA_CONTENT = "schema-content: any string is ok"
)

func (this *worker_V1) Schema() tssd.Schema {
	fmt.Println("worker_V1 types:", this.Types())
	ret := string(tssd.HashFunc(this.Types()))
	fmt.Println("hash value:", ret)
	return tssd.Schema{
		-1,
		//string(tssd.HashFunc(this.Types())),
		ret,
		this.TID(),
		SCHEMA_CONTENT,
	}
}

// test V1->V2
func TestUnmarshalDecorateWorker(t *testing.T) {
	//make sure to regist current firstly
	tssd.Register(&worker{})
	tssd.Register(&worker_V1{})

	st := worker_V1{
		Name: name,
		//"White Hourse",
		Age: 80,
	}

	buf, _ := tssd.Marshal(&st)

	fmt.Println("st buf: ", buf.Fragments()[0].Data)

	var s1 worker_V1
	buf2 := tssd.Pipe(buf)
	//buf input by v1, you can receive v1
	err := tssd.UnmarshalTo(buf2, &s1)
	if err != nil || s1.Name != name || s1.Age != age {
		t.Errorf("unmarshalTo v1 fail")
	}

	fmt.Println("v1: ", s1)

	var s2 worker_V2
	tssd.Register(&worker_V2{})
	//buf input by v1, you can receive v2
	err = tssd.UnmarshalTo(buf2.Rewind(), &s2)
	if err != nil || s2.Name != name || s2.Age != age || s2.Address != defaultAddress {
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}
}

// V2->V1(fail), V2->V1
func TestUnmarshalDecorateWorker2(t *testing.T) {

	//just register once, so put them in init() is good choice
	tssd.Register(&worker{})
	tssd.Register(&worker_V1{})

	st := worker_V2{
		Name:    name,
		Address: "White Hourse",
		Age:     age,
	}

	buf, _ := tssd.Marshal(&st)

	var s1 worker_V1
	buf2 := tssd.Pipe(buf)
	//buf input by v2, you can't downgrade to v1
	err := tssd.UnmarshalTo(buf2, &s1)
	if err == nil {
		t.Errorf("unmarshalTo v1  should fail")
	}

	var s2 worker_V2
	//buf input by v1, you can receive v2
	err = tssd.UnmarshalTo(buf2.Rewind(), &s2)
	if err != nil || s2.Name != name || s2.Age != age || s2.Address != st.Address {
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}
}

type TestStruct struct {
	Name string
	tssd.Flat[TestStruct, *TestStruct]
	Age int16
}

func (this *TestStruct) Group() string {
	return "TestStruct"
}

func (this *TestStruct) Version() string {
	return "TestStruct"
}

type TestStruct2 struct {
	TestStruct
	I int8
	tssd.Flat[TestStruct2, *TestStruct2]
}

func (this *TestStruct2) Group() string {
	return "TestStruct2"
}

func (this *TestStruct2) Version() string {
	return "TestStruct2"
}

type TestStruct3 struct {
	TestStruct
	tssd.Flat[TestStruct3, *TestStruct3]
}

func (this *TestStruct3) Group() string {
	return "TestStruct3"
}

func (this *TestStruct3) Version() string {
	return "TestStruct3"
}

func TestTypesExcludeFlat(t *testing.T) {
	tssd.Register(&worker_V1{})
	tssd.Register(&TestStruct{})
	tssd.Register(&TestStruct2{})
	tssd.Register(&TestStruct3{})

	v1 := worker_V1{}

	expect := []int8{tssd.Tobject, 2, 0, tssd.Tstring, tssd.Tint16}
	if !tssd.TypesEqual(v1.Types(), expect) {
		t.Errorf("TestTypesExcludeFlat v1 fail")
	}

	v2 := TestStruct{}
	if !tssd.TypesEqual(v2.Types(), expect) {
		t.Errorf("TestTypesExcludeFlat test2 fail")
	}

	v3 := TestStruct2{}
	t3 := v3.Types()
	expect2 := []int8{tssd.Tobject, 2, 0, tssd.Tobject, 2, 0, tssd.Tstring, tssd.Tint16, tssd.Tint8}
	if !tssd.TypesEqual(t3, expect2) {
		fmt.Println("TestTypesExcludeFlat: ", t3)
		t.Errorf("TestTypesExcludeFlat test3 fail")
	}

	v4 := TestStruct3{}
	t4 := v4.Types()
	expect3 := []int8{tssd.Tobject, 1, 0, tssd.Tobject, 2, 0, tssd.Tstring, tssd.Tint16}
	if !tssd.TypesEqual(t4, expect3) {
		fmt.Println("TestTypesExcludeFlat: ", t4)
		t.Errorf("TestTypesExcludeFlat test4 fail")
	}
}
