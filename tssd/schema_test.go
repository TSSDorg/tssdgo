package tssd_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	tssd "github.com/tssdorg/tssdgo/tssd"
)

//this file demo how to override Flatable.Schema()

const WORKER_GROUP_NAME = "work_group_name"

//you can alias to simplify for users,
//but update it after every update the struct
type worker = worker_V2

////////////////////////////the worker V2////////////////////////////////////
//after you need add/update the struct
//you need implment Progeny() string to specify which version
//and rename a new class name
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

//generate an uniqu id
func (this *worker_V2) TID() string {
	return "worker_V2-" + strconv.Itoa(tid)
}

func (this *worker_V2) Schema() tssd.Schema {
	return tssd.Schema{
		this.Hash(this.Types()),
		this.TID(),
		-1,
		"you can put a json object string",
	}
}

func (this *worker_V2) OnHeader(header tssd.Header) (err error) {
	//TSSD format version may update in future, do some check here
	fmt.Printf("header version: %d.%d", header.Version[0], header.Version[1])

	schema := header.Schema
	//you can collect all fragments by the tid, fragments, current
	fmt.Printf("tid: %s, fragment id: %d\n", schema.TID, schema.Fragment)

	//and you can get the extend info in header.Schema which from peer
	fmt.Printf("schema hash: [%s] extent content: %s\n", schema.Hash, schema.Extent)
	return nil
}

////////////////////////////the worker V1////////////////////////////////////
//after you need add/update the struct
//you need implment Progeny() string to specify which version
//and rename a new class name
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

//generate an uniqu id
func (this *worker_V1) TID() string {
	tid++
	return strconv.Itoa(tid)
}

const (
	SCHEMA_CONTENT = "schema-content: any string is ok"
)

func (this *worker_V1) Schema() tssd.Schema {
	return tssd.Schema{
		this.Hash(this.Types()),
		this.TID(),
		-1,
		SCHEMA_CONTENT,
	}
}

func (this *worker_V1) OnHeader(header tssd.Header) (err error) {
	//TSSD format version may update in future, do some check here
	fmt.Printf("header version: %d.%d", header.Version[0], header.Version[1])

	schema := header.Schema
	//you can collect all fragments by the tid, fragments, current
	fmt.Printf("tid: %s, fragment id: %d\n", schema.TID, schema.Fragment)

	//and you can get the extend info in header.Schema which from peer
	fmt.Printf("schema hash: [%s] extent content: %s\n", schema.Hash, schema.Extent)
	if header.Schema.Extent != SCHEMA_CONTENT {
		return errors.New("TSSD schema extent info err")
	}
	return nil
}

//test V1->V2
func TestUnmarshalDecorateWorker(t *testing.T) {
	//make sure to regist current firstly
	tssd.Register(&worker{})
	tssd.Register(&worker_V1{})

	st := worker_V1{
		Name: name,
		//"White Hourse",
		Age: age,
	}

	buf, _ := tssd.Marshal(&st)

	fmt.Println("st buf: ", buf)

	var s1 worker_V1
	//buf input by v1, you can receive v1
	err := tssd.UnmarshalTo(buf, &s1)
	if err != nil || s1.Name != name || s1.Age != age {
		t.Errorf("unmarshalTo v1 fail")
	}

	fmt.Println("v1: ", s1)

	var s2 worker_V2
	tssd.Register(&worker_V2{})
	//buf input by v1, you can receive v2
	err = tssd.UnmarshalTo(buf.Rewind(), &s2)
	if err != nil || s2.Name != name || s2.Age != age || s2.Address != defaultAddress {
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}
}

//V2->V1(fail), V2->V1
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
	//buf input by v2, you can't downgrade to v1
	err := tssd.UnmarshalTo(buf, &s1)
	if err == nil {
		t.Errorf("unmarshalTo v1  should fail")
	}

	var s2 worker_V2
	//buf input by v1, you can receive v2
	err = tssd.UnmarshalTo(buf.Rewind(), &s2)
	if err != nil || s2.Name != name || s2.Age != age || s2.Address != st.Address {
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}
}
