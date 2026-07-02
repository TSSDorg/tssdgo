package tssd_test

import (
	"errors"
	"fmt"
	"testing"


	tssd "github.com/tssdorg/tssdgo/tssd"
)

//you can alias to simplify for users, 
//but update it after every update the struct
type worker = worker_V2

////////////////////////////the worker V2////////////////////////////////////
//after you need add/update the struct 
//you need implment Progeny() string to specify which version 
//and rename a new class name
type worker_V2 struct {
	tssd.Flat[worker_V2, *worker_V2]
	Age int16
	Address string  //the new version of worker, which we add a new field Address
	Name string
}

func (this *worker_V2) Version() string {
	return "worker_V2"
}

func (this *worker_V2) Decorate(flat tssd.Flatable) tssd.Flatable{
	old := flat.(*worker_V1)
	this.Name = old.Name
	this.Age = old.Age
	this.Address = defaultAddress
	return this
}

func (this *worker_V2) Progeny() string {
	return "worker_V3"
}

func (this *worker_V2) Schema(factory tssd.Factory) tssd.Schema {
	return tssd.Schema {
		this.Hash(this.Types(factory)),
		"json",  
		"jsonstring",
	}
}

func (this *worker_V2) OnHeader(header tssd.Header) (err error) {
	//
	fmt.Println("header version: {}", header.Version)

	 //and you can get the extend info in header.Schema which from peer
	 fmt.Println("schema type: {}, content: {}", header.Schema.Type, header.Schema.Content)
	 return nil
}

////////////////////////////the worker V1////////////////////////////////////
//after you need add/update the struct 
//you need implment Progeny() string to specify which version 
//and rename a new class name
type worker_V1 struct {
	tssd.Flat[worker_V1, *worker_V1]
	Name string
	Age int16
}

func (this *worker_V1) Version() string {
	return "worker_V1"
}

func (this *worker_V1) Progeny() string {
	return "worker_V2"
}

const (
	SCHEMA_TYPE = "schema-type as you want"
	SCHEMA_CONTENT = "schema-content: any string is ok"
)

func (this *worker_V1) Schema(factory tssd.Factory) tssd.Schema {
	return tssd.Schema {
		this.Hash(this.Types(factory)),
		SCHEMA_TYPE,  
		SCHEMA_CONTENT,
	}
}

func (this *worker_V1) OnHeader(header tssd.Header) (err error) {
	 //and you can get the extend info in header.Schema which from peer
	 //maybe it's "blabla.."
	 fmt.Println("schema type: {}, content: {}", header.Schema.Type, header.Schema.Content)
	 if header.Schema.Type != SCHEMA_TYPE || header.Schema.Content != SCHEMA_CONTENT {
		return errors.New("TSSD schema extent info err")
	 }
	 return nil
}

//test V1->V2
func TestUnmarshalDecorateWorker(t *testing.T) {	

	st := worker_V1 {
		Name: name,
		//"White Hourse",
		Age: age,
	}

	buf := marshal(&st)

	//1. user should New a tssd facory with the new version object
	factory := tssd.New(&worker{})

	//2. and register a old version, if you someone may send you a old byte sequence
	//tssd will auto Unmarshal with the old version object and Decorate to return a new object
	factory.Register(&worker_V1{})
	//factory.Register(&worker_V2{})

	var s1 worker_V1
	//buf input by v1, you can receive v1
	_, err := factory.UnmarshalTo(buf, &s1);
	if  err != nil || s1.Name != name || s1.Age != age {
		t.Errorf("unmarshalTo v1 fail")
	}

	fmt.Println("v1: ", s1)

	var s2 worker_V2
	//buf input by v1, you can receive v2
	_, err = factory.UnmarshalTo(buf, &s2);
	if  err != nil || s2.Name != name || s2.Age != age || s2.Address != defaultAddress{
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}

	//but you can receive a latest one
	flat, _, err := factory.Unmarshal(buf)
	if  err != nil {
		t.Errorf("unmarshal v3 fail")
	}

	stu, ok := flat.(*worker)
	if !ok || stu.Name != name || stu.Age != age || stu.Address != defaultAddress {
		t.Errorf("unmarshal not Student or failed")
	}

}

//V2->V1(fail), V2->V1
func TestUnmarshalDecorateWorker2(t *testing.T) {	

	st := worker_V2 {
		Name: name,
		Address: "White Hourse",
		Age: age,
	}

	buf := marshal(&st)

	//1. user should New a tssd facory with the new version object
	factory := tssd.New(&worker{})

	//2. and register a old version, if you someone may send you a old byte sequence
	//tssd will auto Unmarshal with the old version object and Decorate to return a new object
	factory.Register(&worker_V1{})
	factory.Register(&worker_V2{})

	var s1 worker_V1
	//buf input by v2, you can't downgrade to v1
	_, err := factory.UnmarshalTo(buf, &s1);
	if  err == nil {
		t.Errorf("unmarshalTo v1  should fail")
	}

	var s2 worker_V2
	//buf input by v1, you can receive v2
	_, err = factory.UnmarshalTo(buf, &s2);
	if  err != nil || s2.Name != name || s2.Age != age || s2.Address != st.Address {
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}


	//but you can receive a latest one
	flat, _, err := factory.Unmarshal(buf)
	if  err != nil {
		t.Errorf("unmarshal v3 fail")
	}

	stu, ok := flat.(*worker)
	if !ok || stu.Name != name || stu.Age != age || stu.Address != st.Address {
		t.Errorf("unmarshal not Student or failed")
	}
}
