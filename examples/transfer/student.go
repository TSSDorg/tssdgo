package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	tssd "github.com/tssdorg/tssdgo/tssd"
)

type Course struct {
	Name     string
	TestTime time.Time
	Score    float32
}

type School struct {
	Name           string
	Camp           []string
	EntryLeaveTime [2]time.Time
}

type Student struct {
	tssd.Flat[Student, *Student]    // add Flat Base here
	ID      int64
	Name    []string
	Age     uint8
	Value   float64
	Levels  []int
	IsMale  bool
	Birth   time.Time
	Address []string
	Mail    string
	Schools []School
	Courses map[string]Course
}

const STUDENT_GROUP = "Student"
// Version and Group is the two method you should implement
func (this *Student) Version() string { return "V1" }
func (this *Student) Group() string   { return STUDENT_GROUP }

//make sure register before Marshal or Unmarshal
func init() {
	tssd.Register(&Student{})
}

func handleSchemaRecv(rw io.ReadWriter) *tssd.Schema {
	bs := make([]byte, 1024)
	nbuf := &tssd.Buffer{}
	for {
		n, err := rw.Read(bs)
		if err != nil || n == 0 {
			fmt.Println("Error occurred while reading:", err)
			return nil
		}
		nbuf.Append(bs[:n])

		schema := &tssd.Schema{}
		if err := schema.Unmarshal(nbuf); err == nil {
			return schema
		}
	}
}

var nbuf *tssd.Buffer

func handleEcho(rw io.ReadWriter) error {
	now := time.Now()
	if nbuf == nil {
		nbuf = &tssd.Buffer{
			MTU: 256,
		}
		v := &Student{
			ID:      101,
			Name:    []string{"Tom", "W", "Bush"},
			Value:   98.5,
			Levels:  []int{6, 7, 9, 8, 10},
			Age:     22,
			Birth:   now.AddDate(-22, 0, 0),
			IsMale:  true,
			Address: []string{"5th street 11", "1st road 123"},
			Mail:    "tom@gmail.com",
			Courses: map[string]Course{
				"phisic":  {Name: "phisic", TestTime: now.AddDate(0, -5, 0), Score: 80.5},
				"english": {Name: "english", TestTime: now.AddDate(0, -2, 0), Score: 93.8},
			},
			Schools: []School{
				{Name: "1st jounir school", Camp: []string{"1", "2"}, EntryLeaveTime: [2]time.Time{now.AddDate(-6, 0, 0), now.AddDate(-3, 0, 0)}},
				{Name: "primary school", Camp: []string{"23", "456"}, EntryLeaveTime: [2]time.Time{now.AddDate(-3, 0, 0), now.AddDate(0, -1, 0)}},
			},
		}

		// marshal into a Buffer
		// then you got the data:  Buffer.Fragments[i].Data
		if err := tssd.MarshalTo(v, nbuf); err != nil {
			fmt.Println("Error occurred while marshalling:", err)
			return err
		}
	}

	schema := handleSchemaRecv(rw)
	if schema == nil {
		fmt.Println("Error occurred while reading schema")
		return errors.New("failed to receive schema")
	}
	fmt.Println("Received schema:", schema)

	switch {
	case schema.Fragment <= 0:

		// get the data from Buffer.Fragments[i].Data
		for i := 0; i < len(nbuf.Fragments); i++ {
			n, err := rw.Write(nbuf.Fragments[i].Data)
			fmt.Println("writing:", err, n, nbuf.Size, len(nbuf.Fragments[i].Data), nbuf.Fragments[i].Data)
		}
	case schema.Fragment > 0 && schema.Fragment <= int16(len(nbuf.Fragments)):
		rw.Write(nbuf.Fragments[schema.Fragment-1].Data)
	default:
		fmt.Println("Invalid fragment number:", schema.Fragment)
	}
	return nil
}

// client
func handleFragmentRecv(rw io.ReadWriter, bs []byte) ([]byte, *tssd.Fragment) {
	var b [1024]byte
	frag := &tssd.Fragment{}
	var err error
	for {
		if len(bs) > 0 {
			// read or recv Data, then Fragment.Unmarshal
			bs, err = frag.Unmarshal(bs)
			if err == nil {
				fmt.Println("Received fragment:", frag.Fragment, " with length:", len(frag.Data))
				return bs, frag
			}
			if err != nil && errors.Is(err, tssd.ErrorInSufficientData) {
				fmt.Println("Error occurred while unmarshalling:", err)
				return bs, nil
			}
		}
		n, err := rw.Read(b[:])
		if err != nil || n == 0 {
			fmt.Println("Error occurred while reading:", err)
			return bs, nil
		}
		bs = append(bs, b[:n]...)
	}
}

func query(rw io.ReadWriter) {
	schema := &tssd.Schema{}

	cBuf := &tssd.Buffer{}
	schema.Marshal(cBuf)
	rw.Write(cBuf.Fragments[0].Data)

	fBuf := &tssd.Buffer{}

	bs := make([]byte, 0, 1024)
	frag := &tssd.Fragment{}
	// Buffer.Wanted will return the Fragment missing
	// or return 0 if Fragments complete
	for fBuf.Wanted() > 0 {
		bs, frag = handleFragmentRecv(rw, bs)
		if frag == nil {
			fmt.Println("Error occurred while receiving fragment")
			return
		}
		// receive and unmarshal fragment, then push into a Buffer
		fBuf.Push(frag)
	}

	// When Buffer is complete, you can Unmarshal to a object
	var stu Student
	if err := tssd.UnmarshalTo(fBuf, &stu); err != nil {
		fmt.Println("Error occurred while unmarshalling:", err)
		return
	}

	fmt.Println("Received full student:", stu)

	// if you find Fragment damaged, query it again
	// we simple send a schema, you should design your protocol to interaction with you server
	schema.Fragment = 2
	schema.Marshal(cBuf.Clear())
	rw.Write(cBuf.Fragments[0].Data)
	fmt.Println("send schema:", cBuf.Fragments[0].Data)

	bs, frag = handleFragmentRecv(rw, bs)
	if frag == nil {
		fmt.Println("Error occurred while receiving fragment")
		return
	}
	fBuf.Push(frag)

	var stu2 Student
	if err := tssd.UnmarshalTo(fBuf, &stu2); err != nil {
		fmt.Println("Error occurred while unmarshalling:", err)
		return
	}

	fmt.Println("Received full student2:", stu2)

	return
}
