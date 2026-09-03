package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	tssd "github.com/tssdorg/tssdgo/tssd"
)

type Course struct {
	Title   string
	Teacher string
	Score   float32
}

type Contact struct {
	Name     string
	Relation string
	Phone    string
	Address  string
}

type Paper struct {
	Title   string
	Tags    []string
	Content string
}

type Student struct {
	tssd.Flat[Student, *Student] // add Flat Base here

	ID     uint64
	Name   string
	Age    int16
	IsMale bool
	Birth  time.Time

	Contacts []Contact
	Courses  map[string]Course
	Papers   []Paper
}

const STUDENT_FAMILY = "Student"

// Version and Family is the two method you should implement
func (this *Student) Version() string { return "V1" }
func (this *Student) Family() string  { return STUDENT_FAMILY }

// demo: simple request with a fragment
type Request struct {
	tssd.Flat[Request, *Request]
	Fid   int16
	Types string
	Tid   string
	Time  time.Time
	// bla, bla, maybe you need send something others
}

func (this *Request) Version() string { return "V1" }
func (this *Request) Family() string  { return "Request" }

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

	list := rBuf.Fragments()
	fmt.Println("recv Request: ", list[0].Data)

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
	if nbuf == nil {
		nbuf = &tssd.Buffer{
			MTU: 512,
		}
		v := &Student{
			ID:     101,
			Name:   "Donald J Tramp",
			Age:    80,
			IsMale: true,
			Birth:  time.Now().AddDate(-22, 1, 2),
			Contacts: []Contact{
				Contact{
					Name:     "Alice",
					Relation: "Mother",
					Phone:    "13456778889966677788",
					Address:  "werhweuirhewirhierhiehiehterihtre",
				},
				Contact{
					Name:     "Bob",
					Relation: "Father",
					Phone:    "13556778889966677788",
					Address:  "afwererewerhweuirhewirhierhiehiehterihtre",
				},
			},
			Courses: map[string]Course{
				"English": Course{
					Title:   "English",
					Teacher: "Mrs White",
					Score:   90.5,
				},
				"Math": Course{
					Title:   "Math",
					Teacher: "Mr. Frank",
					Score:   80.5,
				},
			},
			Papers: []Paper{
				Paper{
					Title:   "xxxxxxxxxxxxxxxx study",
					Tags:    []string{"AI", "Math", "algorithem"},
					Content: "asfdsfwererrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrweetertertrtr",
				},
				Paper{
					Title:   "xxxxxxxxxxxxxxxxyyy study",
					Tags:    []string{"AI", "Math", "algorithem"},
					Content: "asfdsfwererrrrrrrdgdgdfgfhfhfrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrweetertertrtr",
				},
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
		return errors.New("failed to receive request")
	}

	switch {
	case request.Fid <= 0:
		// request all fragments
		// you can call Buffer.WriteFragments to write
		n, err := nbuf.WriteFragments(rw)
		fmt.Println("Written bytes:", n, err)
	case request.Fid > 0 && int(request.Fid) <= len(nbuf.Fragments()):
		n, err := nbuf.Fragments()[request.Fid-1].Write(rw)
		// or you can visit nbuf.Fragments()[request.Fid-1].Data and write directly
		// n, err := rw.Write(nbuf.Fragments()[request.Fid-1].Data)
		fmt.Println("written fragment: ", request.Fid, " data len:", n, err)
	default:
		fmt.Println("Invalid fragment number:", request.Fid)
	}
	return nil
}

func sendRequest(fid int, wr io.Writer) error {
	wBuf := &tssd.Buffer{}
	request := &Request{
		Fid: int16(fid),
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
