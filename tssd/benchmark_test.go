package tssd

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"time"
	"testing"
)

type course struct {
	Name string
	TestTime time.Time
	Score float32
}

type school struct {
	Name string
	Camp []string
	EntryLeaveTime [2]time.Time
}

type student struct {
	Flat[student, *student]
	ID    int64
	Name  []string
	Age   uint8
	Value float64
	Levels  []int
	IsMale     bool
	Birth     time.Time
	Address []string
	Mail   string
	Schools []school
	Courses map[string]course
}

func (this *student) Version() string {
	return "V1"
}

func (this *student) Family() string {
	return "STUDENT_FAMILY"
}


var tiStudent *typeInfo
var now = time.Now()
var pStudent = &student {
		ID: 101,
		Name: []string{"Tom", "W", "Bush"},
		Value: 98.5,
		Levels: []int{6, 7, 9, 8, 10},
		Age: 22,
		Birth: now.AddDate(-22, 0, 0),
		IsMale: true,
		Address: []string{"5th street 11", "1st road 123"},
		Mail:  "tom@gmail.com",
		Courses: map[string]course{
			"phisic": {"phisic", now.AddDate(0, -5, 0), 80.5},
			"english": {"english", now.AddDate(0, -2, 0), 93.8},
		},
		Schools: []school{
			{"1st jounir school", []string{"1", "2"}, [2]time.Time{now.AddDate(-6, 0, 0), now.AddDate(-3, 0, 0)}},
			{"primary school", []string{"23", "456"}, [2]time.Time{now.AddDate(-3, 0, 0), now.AddDate(0, -1, 0)}},
		},
	}

func init() {
	tiStudent = parse(student{})
	Register(pStudent)
}

func TestStudentStorageSize(t *testing.T) {
	blob, _ := json.Marshal(*pStudent)
	fmt.Println("json size:", len(blob))

	var network bytes.Buffer
	gob.NewEncoder(&network).Encode(pStudent)
	fmt.Println("gob size:", len(network.Bytes()))
	buf, _ := tiStudent.marshal(pStudent)
	fmt.Println("tssd size:", buf.Size)

	n := &Buffer {MTU: 2048}

	MarshalTo(pStudent, n)
	fmt.Println("tssd Fragments size:", len(n.fragments[0].Data))
}


func BenchmarkTypeInfoMarshal(b *testing.B) {
	buf := &Buffer {
		MTU: 2048,
	}
	for i := 0; i < b.N; i++ {
		tiStudent.marshalTo(pStudent, buf.Clear())
	}
}

func BenchmarkGobMarshal(b *testing.B) {
	var network bytes.Buffer
	for i := 0; i < b.N; i++ {
		network.Reset()
		gob.NewEncoder(&network).Encode(pStudent)
	}
}

func BenchmarkJsonMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json.Marshal(*pStudent)
	}
}


func BenchmarkTypeInfoUnmarshal(b *testing.B) {
	buf, _ := tiStudent.marshal(pStudent)
	var s student
	for i := 0; i < b.N; i++ {
		buf.Rewind()
		if err := tiStudent.unmarshalTo(buf, &s); err != nil {
			b.Error("Failed to unmarshal student: ", err)
		}
	}
}

func BenchmarkGobUnmarshal(b *testing.B) {
	var network bytes.Buffer
	var s student
	gob.NewEncoder(&network).Encode(pStudent)
	for i := 0; i < b.N; i++ {
		gob.NewDecoder(bytes.NewReader(network.Bytes())).Decode(&s)
	}
}

func BenchmarkJsonUnmarshal(b *testing.B) {
	blob, _ := json.Marshal(*pStudent)
	var s student
	for i := 0; i < b.N; i++ {
		json.Unmarshal(blob, &s)
	}
}

func BenchmarkTSSDMarshal(b *testing.B) {
	n := &Buffer {MTU: 2048}
	for i := 0; i < b.N; i++ {
		MarshalTo(pStudent, n.Clear())
	}
}

func BenchmarkTSSDUUnmarshal(b *testing.B) {
	n := &Buffer {}
	MarshalTo(pStudent, n)
	var s student
	for i := 0; i < b.N; i++ {
		n.Rewind()
		UnmarshalTo(n, &s)
	}
}
