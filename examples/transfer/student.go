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
	tssd.Flat[Student, *Student] // add Flat Base here
	ID                           int64
	Name                         []string
	Age                          uint8
	Value                        float64
	Levels                       []int
	IsMale                       bool
	Birth                        time.Time
	Address                      []string
	Mail                         string
	Schools                      []School
	Courses                      map[string]Course
}

const STUDENT_GROUP = "Student"

// Version and Group is the two method you should implement
func (this *Student) Version() string { return "V1" }
func (this *Student) Group() string   { return STUDENT_GROUP }

// demo: simple request with a fragment
type Request struct {
	tssd.Flat[Request, *Request]
	Fid int32
	// bla, bla, maybe you need send something others
}

func (this *Request) Version() string { return "V1" }
func (this *Request) Group() string   { return "Request" }

// make sure register before Marshal or Unmarshal
func init() {
	tssd.Register(&Student{})
	tssd.Register(&Request{})
}

func handleRequestRecv(rr io.Reader) (*Request, error) {
	rBuf := &tssd.Buffer{}

	if err := rBuf.ReadFragments(rr); err != nil {
		fmt.Println("Error occurred while read unmarshalling fragments:", err)
		return nil, err
	}

	// When Buffer is complete, you can Unmarshal to a object
	var req Request
	if err := tssd.UnmarshalTo(rBuf, &req); err != nil {
		fmt.Println("Error occurred while unmarshalling:", err)
		return nil, err
	}

	fmt.Println("Received full student:", req)
	return &req, nil
}

var nbuf *tssd.Buffer

func handleEchoRequest(rw io.ReadWriter) error {
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
		// then you got the data:  Buffer.Fragments()[i].Data
		if err := tssd.MarshalTo(v, nbuf); err != nil {
			fmt.Println("Error occurred while marshalling:", err)
			return err
		}
	}

	request, err := handleRequestRecv(rw)
	if err != nil || request == nil {
		fmt.Println("Error occurred while reading request")
		return errors.New("failed to receive schema")
	}

	switch {
	case request.Fid <= 0:
		// request all fragments
		// get the data from Buffer.Fragments()[i].Data
		for i := 0; i < len(nbuf.Fragments()); i++ {
			n, err := rw.Write(nbuf.Fragments()[i].Data)
			fmt.Println("writing:", err, n, nbuf.Size, len(nbuf.Fragments()[i].Data), nbuf.Fragments()[i].Data)
		}
	case request.Fid > 0 && int(request.Fid) <= len(nbuf.Fragments()):
		n, err := rw.Write(nbuf.Fragments()[request.Fid-1].Data)
		fmt.Println("written fragment: ", request.Fid, " data len:", n, err)
	default:
		fmt.Println("Invalid fragment number:", request.Fid)
	}
	return nil
}

func sendRequest(fid int, wr io.Writer) error {
	wBuf := &tssd.Buffer{}
	request := &Request{
		Fid: int32(fid),
	}
	// marshal request into a Buffer
	// then you got the data:  Buffer.Fragments()[i].Data
	if err := tssd.MarshalTo(request, wBuf); err != nil {
		fmt.Println("Error occurred while marshalling request:", err)
		return err
	}

	n, err := wBuf.WriteFragments(wr)
	if err != nil {
		fmt.Println("Write request err:", err, ", written:", n)
		return err
	}
	fmt.Println("Write request written:", n)
	return nil
}

func query(rw io.ReadWriter) {

	sendRequest(0, rw)

	rBuf := &tssd.Buffer{}
	if err := rBuf.ReadFragments(rw); err != nil {
		fmt.Println("Error occurred while read unmarshalling fragments:", err)
		return
	}

	// When Buffer is complete, you can Unmarshal to a object
	var stu Student
	if err := tssd.UnmarshalTo(rBuf, &stu); err != nil {
		fmt.Println("Error occurred while unmarshalling:", err)
		return
	}

	fmt.Println("Received full student:", stu)

	// if you find Fragment damaged, query it again
	// we request Fid 2
	sendRequest(2, rw)

	frag := new(tssd.Fragment)
	//bio may remain some data thtat we need
	//so reuse the previous one
	if err := frag.Read(rw); err != nil {
		return
	}

	//push again
	if _, err := rBuf.Push(frag); err != nil {
		return
	}

	var stu2 Student
	if err := tssd.UnmarshalTo(rBuf, &stu2); err != nil {
		fmt.Println("Error occurred while unmarshalling:", err)
		return
	}

	fmt.Println("Received full student2:", stu2)

	return
}
