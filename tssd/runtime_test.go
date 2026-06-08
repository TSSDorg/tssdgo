package tssd_test


import (
    "fmt"
    "unsafe"
    "testing"
    "github.com/tssdorg/tssdgo/tssd"
    )

    
//actually mem depend on unsafe deeply
//and below memory skills which is not guaraent by Go project
//so we should test the compation before update Go tools/runtime

func TestAnySliceToByteSlice(t *testing.T) {
    str := []string{"hello", "world"} 
    fmt.Println(len(str))
    //Output:2
    
    p := (*[]byte)(tssd.Ptr(&str))
    fmt.Println(len(*p))
    //Output:2
    
    if len(str) != len(*p) {
        t.Error("TestAnySliceToByteSlice failed")
    }
}


func TestArrayAddr(t *testing.T) {
    str := [2]string{"hello", "world"}
    if tssd.Ptr(&str) != tssd.Ptr(&str[0]) {
        t.Error("TestArrayAddr failed 1")
    }
    
    STRING_ARRAY_ADDR_GAP := uintptr(tssd.Ptr(&str[1])) - uintptr(tssd.Ptr(&str[0]))
    
    p := (*[1]string)(tssd.Ptr(uintptr(tssd.Ptr(&str[0])) + STRING_ARRAY_ADDR_GAP))
    
    //fmt.Printf("p, %s, %d; str1: %s %d\n", *p, len(*p), str[1], len(str[1]))
    
    if len((*p)[0]) != len(str[1]) { t.Error("TestArrayAddr failed 2") }
    
    for i:=0; i<len(str[1]); i++ {
        if (*p)[0][i] != str[1][i] { 
           t.Error("TestArrayAddr failed 2")
        }
    }
}

func TestArrayToSlice(t *testing.T) {
    
    var arr [5]byte = [5]byte{0, 1, 2, 3, 4}
    
    var slc []byte = arr[0:5]
    
    for i:=0;  i<len(arr); i++ {
        if tssd.Ptr(&arr[i]) != tssd.Ptr(&slc[i]) {
            t.Error("TestArrayToSlice failed")    
        }
    }
}


func TestAnyToByteSlice(t *testing.T) {
    
    var x int32 = 0x01023456
    
    p := (*[unsafe.Sizeof(x)]byte)(tssd.Ptr(&x))
    
    b := (*p)[0:unsafe.Sizeof(x)]
    
    fmt.Printf("x: %x, b: %x %x %x %x\n", x, b[0], b[1], b[2], b[3])
    fmt.Println("b: ", b)
    
    if b[0]!=0x56 || b[1]!=0x34 || b[2]!=0x02 || b[3]!=0x01 {
        t.Error("TestAnyToByteSlice failed")
    }
}

func TestMakeString(t *testing.T) {
    var str string
    p := tssd.Ptr(&str)
    s := (*string)(p)
    *s = string("hello w")
    
    if str != "hello w" {
        t.Error("TestMakeString failed")
    }
}


func TestMakeStringSlice(t *testing.T) {
    var sstr []string
    p := tssd.Ptr(&sstr)
    s := (*[]string)(p)
    *s = make([]string, 2)
    
    (*s)[0] = "hello"
    (*s)[1] = "world"
    
    if sstr[0] != "hello" || sstr[1] != "world" {
        t.Error("TestMakeString failed")
    }
}

func TestMakeSlice(t *testing.T) {
    var src []int32 = []int32{100, 101}
    var dest []int32
    l := 8 //unsafe.Sizeof(i32[0]) * len(i32)
    
    p := tssd.Ptr(&dest)
    ds := (*[]byte)(p)
    *ds = make([]byte, l)  //alloc mem
    *ds = (*ds)[0:2]      //reset size
        
     p = tssd.Ptr(&src)
     s := (*[]byte)(p)
     
    copy(tssd.Slice(tssd.Ptr(&(*ds)[0]),  tssd.Size_t(l)),  tssd.Slice(tssd.Ptr(&(*s)[0]),  tssd.Size_t(l)))
    
    if len(dest) != len(src) || dest[0] != src[0] || dest[1]!=src[1] {
        t.Error("TestMakeSlice failed")
    }
}

func TestMakeStructSlice(t *testing.T) {
    type st struct {
        id int
        name string
    }
    
    var src, dest []st
    src = append(src, st{id:100, name:"ello"})
    src = append(src, st{id:100, name:"word"})
    
    gap := unsafe.Sizeof(src[0])
    
    p := tssd.Ptr(&dest)
    ds := (*[]byte)(p)
    *ds = make([]byte, int(gap)*2)  //alloc mem
    *ds = (*ds)[0:2]      //reset size
    
    pds := (*[]st)(tssd.Ptr(ds))
    
    (*pds)[0].id = src[0].id
    (*pds)[1].id = src[1].id
    
    (*pds)[0].name = src[0].name
    (*pds)[1].name = src[1].name
    
    if len(dest) != len(src) || dest[0] != src[0] || dest[1]!=src[1] {
        t.Error("TestMakeStructSlice failed")
    }
}
/*
func TestMapLen(t *testing.T) {
    m1 := map[int]string {
        1 : "abc",
        2 : "cde",
    }
    
    p :=  (*map[byte]byte)(tssd.Ptr(&m1))
    
    if len(m1) != len(*p)  {
        t.Error("TestMapLen failed")
    }
}
*/
func TestPtrToSlice(t *testing.T) {
    var ui uint32 = 0x12345678
    bs := tssd.Slice(tssd.Ptr(&ui), 4)
    if len(bs) != 4 || bs[0] != 0x78 || bs[1] != 0x56 || bs[2] != 0x34 || bs[3] != 0x12 {
        t.Error("Testtssd.PtrToSlice failed")
    }
}
