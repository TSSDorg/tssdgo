# TSSDGo
[TSSD](http://https://github.com/TSSDorg/TSSD-Spec "TSSD") is an open binary data for exchange or storage.
tssdgo implement TSSD with Go(golang), you can read, write and print TSSD data with this go package easily.
## features

- **API simple** **effective**: parse with reflect once, run without reflect(except map) 
- **schema validation**: validate schema to prevent crash caused by unmashaling unmatched data
- **struct migration support**: support receive old struct data and migration to the latest version
- **less depenency**: depend github.com/eineder/printtree only, which for print TSSD data

## quick start

```
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

//marshal is a TSSD writer
//you can storage or send to others
func marshal(obj tssd.Flatable) []byte {
	//factory is a TSSD encoder
	//normal you should run it in init()
	//and save it to reuse
	factory := tssd.New(obj)

	//dest, err := facoty.Marshal(&st)
	//OR you would like within your space
	buf, _ := factory.MarshalTo(obj, make([]byte, 0, 4096))
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

	//got some TSSD data from client
	buf := marshal(&st)
	
	//1. user should New a tssd facory with the new version object
	factory := tssd.New(&student{})

	//2. and register a old version, if you someone may send you a old data struct
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
```
### limitation

1. Big Endian platform in development
2. Go's map can't addressable, so it need reflect when marshal/unmarshal reflect
