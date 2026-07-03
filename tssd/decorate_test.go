package tssd_test

import (
	"fmt"
	"testing"

	tssd "github.com/tssdorg/tssdgo/tssd"
)

var studentFactory *tssd.Factory

func init() {
	studentFactory = tssd.New(&student{})
}

//you can alias to simplify for users, 
//but update it after every update the struct
type student = student_V3

////////////////////////////the student V3////////////////////////////////////
//after you need add/update the struct 
//you need implment Progeny() string to specify which version 
//and rename a new class name
type student_V3 struct {
	tssd.Flat[student_V3, *student_V3]
	Address []string  //the new version of student, which we update to slice
	Age int16
	Name string
}

func (this *student_V3) Version() string {
	return "student_V3"
}

func (this *student_V3) Decorate(flat tssd.Flatable) tssd.Flatable{
	//you may upgrade from v2
	if old, ok := flat.(*student_V2); ok {
		this.Name = old.Name
		this.Age = old.Age
		this.Address = append(this.Address, old.Address)
	}
	//you may upgrade from v1
	if old, ok := flat.(*student_V1); ok {
		this.Name = old.Name
		this.Age = old.Age
		this.Address = append(this.Address, defaultAddress)
	}
	return this
}

////////////////////////////the student V2////////////////////////////////////
//after you need add/update the struct 
//you need implment Progeny() string to specify which version 
//and rename a new class name
type student_V2 struct {
	tssd.Flat[student_V2, *student_V2]
	Age int16
	Address string  //the new version of student, which we add a new field Address
	Name string
}

func (this *student_V2) Version() string {
	return "student_V2"
}

func (this *student_V2) Decorate(flat tssd.Flatable) tssd.Flatable{
	old := flat.(*student_V1)
	this.Name = old.Name
	this.Age = old.Age
	this.Address = defaultAddress
	return this
}

func (this *student_V2) Progeny() string {
	return "student_V3"
}

////////////////////////////the student V1////////////////////////////////////
//after you need add/update the struct 
//you need implment Progeny() string to specify which version 
//and rename a new class name
type student_V1 struct {
	tssd.Flat[student_V1, *student_V1]
	Name string
	Age int16
}

func (this *student_V1) Version() string {
	return "student_V1"
}

func (this *student_V1) Progeny() string {
	return "student_V2"
}


var name = "Donald J. Trump"
var age int16 = 80
var defaultAddress = "White House"

func marshal(obj tssd.Flatable) []byte {

	factory := tssd.New(obj)

	//dest, err := facoty.Marshal(&st)

	//OR you would like within your space
	//dest := make([]byte, 1024)
	buf, _ := factory.MarshalTo(obj, make([]byte, 0, 4096))
	fmt.Println("marshal:", buf, factory)

	//save or send to another space
	//saveOrSend(dest)
	return buf
}

//test V1->V2->V3
func TestUnmarshalDecorate(t *testing.T) {	

	st := student_V1 {
		Name: name,
		//"White Hourse",
		Age: age,
	}

	buf := marshal(&st)

	//1. user should New a tssd facory with the new version object
	factory := studentFactory //tssd.New(&student{})

	//2. and register a old version, if you someone may send you a old byte sequence
	//tssd will auto Unmarshal with the old version object and Decorate to return a new object
	factory.Register(&student_V1{})
	factory.Register(&student_V2{})

	var s1 student_V1
	//buf input by v1, you can receive v1
	_, err := factory.UnmarshalTo(buf, &s1);
	if  err != nil || s1.Name != name || s1.Age != age {
		t.Errorf("unmarshalTo v1 fail")
	}

	fmt.Println("v1: ", s1)

	var s2 student_V2
	//buf input by v1, you can receive v2
	_, err = factory.UnmarshalTo(buf, &s2);
	if  err != nil || s2.Name != name || s2.Age != age || s2.Address != defaultAddress{
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}

	var s3 student
	_, err = factory.UnmarshalTo(buf, &s3);
	if  err != nil || s3.Name != name || s3.Age != age || s3.Address[0] != defaultAddress {
		t.Errorf("unmarshalTo v3 fail")
	}

	//but you can receive a latest one
	flat, _, err := factory.Unmarshal(buf)
	if  err != nil {
		t.Errorf("unmarshal v3 fail")
	}

	stu, ok := flat.(*student)
	if !ok || stu.Name != name || stu.Age != age || stu.Address[0] != defaultAddress {
		t.Errorf("unmarshal not Student or failed")
	}

}

func TestObjectPtr(t *testing.T) {
	st := student {
		Name: name,
		//"White Hourse",
		Age: age,
	}

	fmt.Printf("%p %p %p\n", &st, &st.Flat, &st.Name)

	fmt.Println("st Name:", st.Version())

	//tssd.Parse(st)
}

//V2->V1(fail), V2->V3 ok
func TestUnmarshalDecorate2(t *testing.T) {	

	st := student_V2 {
		Name: name,
		Address: "White Hourse",
		Age: age,
	}

	buf := marshal(&st)

	//1. user should New a tssd facory with the new version object
	factory := tssd.New(&student{})

	//2. and register a old version, if you someone may send you a old byte sequence
	//tssd will auto Unmarshal with the old version object and Decorate to return a new object
	factory.Register(&student_V1{})
	factory.Register(&student_V2{})

	var s1 student_V1
	//buf input by v2, you can't downgrade to v1
	_, err := factory.UnmarshalTo(buf, &s1);
	if  err == nil {
		t.Errorf("unmarshalTo v1  should fail")
	}

	var s2 student_V2
	//buf input by v1, you can receive v2
	_, err = factory.UnmarshalTo(buf, &s2);
	if  err != nil || s2.Name != name || s2.Age != age || s2.Address != st.Address {
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}

	var s3 student
	_, err = factory.UnmarshalTo(buf, &s3);
	if  err != nil || s3.Name != name || s3.Age != age || s3.Address[0] != st.Address {
		t.Errorf("unmarshalTo v3 fail")
	}

	//but you can receive a latest one
	flat, _, err := factory.Unmarshal(buf)
	if  err != nil {
		t.Errorf("unmarshal v3 fail")
	}

	stu, ok := flat.(*student)
	if !ok || stu.Name != name || stu.Age != age || stu.Address[0] != st.Address {
		t.Errorf("unmarshal not Student or failed")
	}
}


//V2->V1(fail), V2->V3 ok
func TestUnmarshalDecorate3(t *testing.T) {	

	st := student_V3 {
		Name: name,
		Address: []string {"White Hourse",},
		Age: age,
	}

	buf := marshal(&st)

	//1. user should New a tssd facory with the new version object
	factory := tssd.New(&student{})

	//2. and register a old version, if you someone may send you a old byte sequence
	//tssd will auto Unmarshal with the old version object and Decorate to return a new object
	factory.Register(&student_V1{})
	factory.Register(&student_V2{})

	var s1 student_V1
	//buf input by v2, you can't downgrade to v1
	_, err := factory.UnmarshalTo(buf, &s1);
	if  err == nil {
		t.Errorf("unmarshalTo v1  should fail")
	}

	var s2 student_V2
	//buf input by v1, you can receive v2
	_, err = factory.UnmarshalTo(buf, &s2);
	if  err == nil {
		t.Errorf("unmarshalTo v2 should fail")
	}

	var s3 student
	_, err = factory.UnmarshalTo(buf, &s3);
	if  err != nil || s3.Name != name || s3.Age != age || s3.Address[0] != st.Address[0] {
		t.Errorf("unmarshalTo v3 fail")
	}

	//but you can receive a latest one
	flat, _, err := factory.Unmarshal(buf)
	if  err != nil {
		t.Errorf("unmarshal v3 fail")
	}

	stu, ok := flat.(*student)
	if !ok || stu.Name != name || stu.Age != age || stu.Address[0] != st.Address[0] {
		t.Errorf("unmarshal not Student or failed")
	}
}

