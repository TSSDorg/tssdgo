package tssd_test

import (
	"fmt"
	"testing"
	"time"
	tssd "github.com/tssdorg/tssdgo/tssd"
)

//this file test for a complex object Student


type Course struct {
	Name string
	TestTime time.Time
	Score float32
}

func (this Course)Equal(other Course) bool {
	return this.Name == other.Name && this.TestTime.Equal(other.TestTime) && this.Score == other.Score
}

type School struct {
	Name string
	Camp []string
	EntryLeaveTime [2]time.Time
}

func (this *School)Equal(other *School) bool {
	return this.Name == other.Name && tssd.SliceEqual(this.Camp, other.Camp) && 
	this.EntryLeaveTime[0].Equal(other.EntryLeaveTime[0]) && this.EntryLeaveTime[1].Equal(other.EntryLeaveTime[1]) 
}

type Student struct {
	tssd.Flat[Student, *Student]
	ID    int64
	Name  []string 
	Age   uint8
	Value float64
	Levels  []int
	IsMale     bool
	Birth     time.Time
	Address []string
	Mail   string
	Schools []School
	Courses map[string]Course
}

const STUDENT_GROUP = "Student"

func (this *Student) Version() string {
	return "V1"
}

func (this *Student) Group() string {
	return STUDENT_GROUP
}


func (this *Student) Equal(other *Student) bool {
	ret := this.ID == other.ID && this.Age == other.Age && this.Value == other.Value && this.IsMale == other.IsMale && this.Birth.Equal(other.Birth) && this.Mail == other.Mail

	if !ret {
		return false
	}

	if !tssd.SliceEqual(this.Levels, other.Levels) {
		return false
	}

	if !tssd.SliceEqual(this.Address[:], other.Address[:]) {
		return false
	}
	fmt.Println("student Equal 1")

	if len(this.Schools) != len(other.Schools) {
		return false
	}
	for i := range len(this.Schools) {
		if !(&this.Schools[i]).Equal(&other.Schools[i]) {
			return false
		}
	}
	fmt.Println("student Equal 2")

	if len(this.Courses) != len(other.Courses) {
		return false
	}
	for k, v := range this.Courses {
		if _, ok := other.Courses[k]; !ok || !v.Equal(other.Courses[k]) {
			fmt.Println("k, v", k, v)
			return false
		}
	}
	fmt.Println("student Equal 3")
	
	return true
}

var now = time.Now()

func TestStudent(t *testing.T) {

	v := Student {
		ID: 101, 
		Name: []string{"Tom", "W", "Bush"}, 
		Value: 98.5, 
		Levels: []int{6, 7, 9, 8, 10},
		Age: 22, 
		Birth: now.AddDate(-22, 0, 0), 
		IsMale: true,
		Address: []string{"5th street 11", "1st road 123"},
		Mail:  "tom@gmail.com",
		Courses: map[string]Course{
			"phisic": {"phisic", now.AddDate(0, -5, 0), 80.5},
			"english": {"english", now.AddDate(0, -2, 0), 93.8},
		},
		Schools: []School{
			{"1st jounir school", []string{"1", "2"}, [2]time.Time{now.AddDate(-6, 0, 0), now.AddDate(-3, 0, 0)}},
			{"primary school", []string{"23", "456"}, [2]time.Time{now.AddDate(-3, 0, 0), now.AddDate(0, -1, 0)}},
		},
	}

	tssd.Register(&v)

	n := &tssd.Buffer {
		MTU: 100,
	}

	tssd.MarshalTo(&v, n)

	if len(n.Fragments[0].Data) == 0 {
		t.Error("TestStruct return row-th failed")
	}

	tssd.Print(&v, *n)
	fmt.Println("fragments:", len(n.Fragments))

	var v2 Student
	tssd.UnmarshalTo(tssd.Pipe(n), &v2)
	fmt.Println("-----v:", v)
	fmt.Println("-----v2:", v2)
	if !v.Equal(&v2) {
		t.Error("TestStruct student failed")
	}

	
	n, _ = tssd.Marshal(&v2)
	if len(n.Fragments[0].Data) == 0 {
		t.Error("TestStruct return row-th 2 failed")
	}

	var v3 Student

	tssd.UnmarshalTo(tssd.Pipe(n), &v3)
	if !v3.Equal(&v) {
		t.Error("TestStruct student failed")
	}

	v2.Address = v2.Address[:0]

	n, _ = tssd.Marshal(&v2)
	if len(n.Fragments[0].Data) == 0 {
		t.Error("TestStruct return row-th 2 failed")
	}

	tssd.UnmarshalTo(tssd.Pipe(n), &v3)
	if !v3.Equal(&v2) {
		t.Error("TestStruct student failed")
	}
}


func TestPrintMap(t *testing.T) {
	type student struct {
		tssd.Flat[student, *student]
		ID    int64
		Levels  []int
		Birth     time.Time
		Address []string
		Courses map[string]Course
	}
	s1 := student {
		ID: 101,
		Levels: []int{6, 7, 9, 8, 10},
		Birth: now.AddDate(-22, 0, 0), 
		Address: []string{"5th street 11", "1st road 123"},
		Courses: map[string]Course{
			"phisi": {"phisi", now.AddDate(0, -5, 0), 80.5},
			"english": {"englis", now.AddDate(0, -2, 0), 93.8},
		},
	}
	
	tssd.Register(&s1)

	n, _ := tssd.Marshal(&s1)

	if len(n.Fragments[0].Data) == 0 {
		t.Error("TestStruct return row-th failed")
	}

	tssd.Print(&s1, *n)
	fmt.Println("Tbase, Tbool, Tstring, Tarray, Tdict, Tobject, Ttime", tssd.Tbase, tssd.Tbool, tssd.Tstring, tssd.Tarray, tssd.Tdict, tssd.Tobject, tssd.Ttime)

	out := []byte{112, 104, 105, 115, 105,}

	fmt.Println("out:", string(out))

	var sout student
	
	tssd.UnmarshalTo(tssd.Pipe(n), &sout)
	fmt.Println(s1)
	fmt.Println(sout)
}

