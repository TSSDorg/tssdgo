# TSSDGo
[TSSD](http://https://github.com/TSSDorg/TSSD-Spec "TSSD") is an open binary data for exchange or storage.
tssdgo implement TSSD with Go(golang), you can read, write and print TSSD data with this go package easily.
## features

- **API simple** **effective**: parse with reflect once, run without reflect(except map)
- **schema validation**: validate schema to prevent crash caused by unmashaling unmatched data
- **struct migration support**: support receive old struct data and migration to the latest version
- **less depenency**: depend github.com/eineder/printtree only, which for print TSSD data


## benchmark test
Marshal
```
$go test -v -bench="(TypeInfo*|Gob*|Json*|TSSD*)Marshal"  
goos: windows
goarch: amd64
pkg: github.com/tssdorg/tssdgo/tssd
cpu: AMD Ryzen 7 8845HS w/ Radeon 780M Graphics
BenchmarkTypeInfoMarshal
BenchmarkTypeInfoMarshal-16       296326              4090 ns/op
BenchmarkGobMarshal
BenchmarkGobMarshal-16            144670              8168 ns/op
BenchmarkJsonMarshal
BenchmarkJsonMarshal-16           298766              4035 ns/op
BenchmarkTSSDMarshal
BenchmarkTSSDMarshal-16           159531              6996 ns/op
```
**TSSD core(TypeInfo 4090) marshal performace is almost same with json(4035),  
TSSD with Fragments support(6996) is little faster than Gob(8168).**

Unmarshal
```
$go test -v -bench="(TypeInfo*|Gob*|Json*|TSSD*)Unmarshal"  
goos: windows
goarch: amd64
pkg: github.com/tssdorg/tssdgo/tssd
cpu: AMD Ryzen 7 8845HS w/ Radeon 780M Graphics
BenchmarkTypeInfoUnmarshal
BenchmarkTypeInfoUnmarshal-16             368180              3337 ns/op
BenchmarkGobUnmarshal
BenchmarkGobUnmarshal-16                   50680             24058 ns/op
BenchmarkJsonUnmarshal
BenchmarkJsonUnmarshal-16                 149484              8010 ns/op
BenchmarkTSSDUnmarshal
BenchmarkTSSDUnmarshal-16                 322578              3356 ns/op
PASS
ok      github.com/tssdorg/tssdgo/tssd  6.018s
```
**TSSD core(TypeInfo 3337) unmarshal performace is 2X fast json(8010),  
TSSD with Fragments support(3356) is 7X faster than Gob(24058).**


## quick start

a simple [transer demo](https://github.com/TSSDorg/tssdgo/blob/main/examples/transfer/student.go)

another support struct version migration snippet

```
//define a const family name for your class/struct
//it will share between versions
const DECORATE_STUDENT_FAMILY = "decorate_test.student"

//you can alias simplify for users
//but update it after every update the struct
type student = student_V2

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

//Family() return the family name
//versions should share the family name, just like the last name in your family
func (this *student_V2) Family() string {
	return DECORATE_STUDENT_FAMILY
}

//Version() should return the uniq version name in family
//just like your first name in your family
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

////////////////////////////the student V1////////////////////////////////////
//after you need add/update the struct
//you need implment Progeny() string to specify which version
//and rename a new class name
type student_V1 struct {
	tssd.Flat[student_V1, *student_V1]
	Name string
	Age int16
}

//Family() return the family name
//versions should share the family name, just like the last name in your family
func (this *student_V1) Family() string {
	return DECORATE_STUDENT_FAMILY
}

func (this *student_V1) Version() string {
	return "student_V1"
}

//Progeny specify which version you will update to
func (this *student_V1) Progeny() string {
	return "student_V2"
}

var name = "Donald J. Trump"
var age int16 = 80
var defaultAddress = "White House"

//0. first of all: regisister all versions of the class
func init() {
	tssd.RegisterCurrent(&student{})
	tssd.Register(&student_V1{})
}

//simple sample within test
func TestUnmarshalDecorate(t *testing.T) {
	st := student_V1 {
		Name: name,
		//"White Hourse",
		Age: age,
	}

	//1. got some TSSD data with Marshal api
	buf, err := tssd.Marshal(&st)

	//2. you can process the TSSD data:  save, send to share etc.
	//socket.Write(buf.Fragments()[0].Data)
	//if you would like to save, call Merge first
	//file.Write(buf.Merge().Fragments()[0].Data)

	//3.  you can Unmarshal the TSSD data to object again after read or receive it
	//buf input by v1, you can receive v1
	var s1 student_V1
	remain, err := factory.UnmarshalTo(buf, &s1);
	if  err != nil || s1.Name != name || s1.Age != age {
		t.Errorf("unmarshalTo v1 fail")
	}

	//4. you can receive v2 from a v1 data, TSSD will auto call Decorate api to migrate
	var s2 student_V2
	remain, err = factory.UnmarshalTo(buf, &s2);
	if  err != nil || s2.Name != name || s2.Age != age || s2.Address != defaultAddress{
		fmt.Println(err, s2)
		t.Errorf("unmarshalTo v2 fail")
	}
}
```
### limitation

1. Big Endian platform in development
2. Go's map can't addressable, so it need reflect when marshal/unmarshal reflect
