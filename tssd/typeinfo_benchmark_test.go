package tssd

import (
	"bytes"
	"encoding/gob"
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
}

func BenchmarkTypeInfoMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tiStudent.marshal(pStudent)
	}
}

func BenchmarkGobMarshal(b *testing.B) {
	var network bytes.Buffer
	for i := 0; i < b.N; i++ {
		network.Reset()
		gob.NewEncoder(&network).Encode(pStudent)
	}
}

func BenchmarkTypeInfoUnmarshal(b *testing.B) {
	buf, _ := tiStudent.marshal(pStudent)
	var s student
	for i := 0; i < b.N; i++ {
		tiStudent.unmarshalTo(buf, &s)
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
