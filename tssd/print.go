package tssd

import (
	"fmt"
	"time"

	print "github.com/eineder/printtree/print"
)

type Node struct {
	Content  string
	Children []*Node
}

func (node *Node) getChildren() []*Node {
	return node.Children
}

func (node *Node) getContent() string {
	return node.Content
}

func (node *Node) print() {
	fmt.Println("node.content:", node.Content)
	for i := 0; i < len(node.Children); i++ {
		node.Children[i].print()
	}
}

func dprintf[T comparable](info *typeInfo, format string, buf []byte) (string, []byte) {
	var d T
	remain, _ := info.dump(info, buf, Ptr(&d))
	return fmt.Sprintf(format, info.name, info.rtype.String(), d), remain
}

func dprintf2[T comparable](info *typeInfo, format string, buf []byte) string {
	var d T
	copy(Slice(Ptr(&d), Size_t(info.size)), buf)
	return fmt.Sprintf(format, d)
}

func parseMergeArray(info *typeInfo, ttype int8, buf []byte) string {
	switch int8(ttype) {
	case Tint8:
		return dprintf2[int8](info, "%d,", buf)
	case Tuint8:
		return dprintf2[uint8](info, "%d,", buf)
	case Tint16:
		return dprintf2[int16](info, "%d,", buf)
	case Tuint16:
		return dprintf2[uint16](info, "%d,", buf)
	case Tint32:
		return dprintf2[int32](info, "%d,", buf)
	case Tuint32:
		return dprintf2[uint32](info, "%d,", buf)
	case Tint64:
		return dprintf2[int64](info, "%d,", buf)
	case Tuint64:
		return dprintf2[uint64](info, "%d,", buf)
	case Tfloat32:
		return dprintf2[float32](info, "%f,", buf)
	case Tfloat64:
		return dprintf2[float64](info, "%f,", buf)
	case Tbool:
		return dprintf2[bool](info, "%t,", buf)
	}
	return ""
}

func (info *typeInfo) parse(parent *Node, buf []byte) []byte {
	if len(buf) == 0 {
		return nil
	}
	if info.Type != int8(buf[0]) && -info.Type != int8(buf[0]) {
		fmt.Println("print parsetype mismatch ", info.Type, buf[0])
		return buf
	}

	node := &Node{}
	parent.Children = append(parent.Children, node)
	switch int8(buf[0]) {
	case Tint8:
		node.Content, buf = dprintf[int8](info, "%s(%s): %d", buf)
	case Tuint8:
		node.Content, buf = dprintf[uint8](info, "%s(%s): %d", buf)
	case Tint16:
		node.Content, buf = dprintf[int16](info, "%s(%s): %d", buf)
	case Tuint16:
		node.Content, buf = dprintf[uint16](info, "%s(%s): %d", buf)
	case Tint32:
		node.Content, buf = dprintf[int32](info, "%s(%s): %d", buf)
	case Tuint32:
		node.Content, buf = dprintf[uint32](info, "%s(%s): %d", buf)
	case Tint64:
		node.Content, buf = dprintf[int64](info, "%s(%s): %d", buf)
	case Tuint64:
		node.Content, buf = dprintf[uint64](info, "%s(%s): %d", buf)
	case Tfloat32:
		node.Content, buf = dprintf[float32](info, "%s(%s): %f", buf)
	case Tfloat64:
		node.Content, buf = dprintf[float64](info, "%s(%s): %f", buf)
	case Tbool:
		node.Content, buf = dprintf[bool](info, "%s(%s): %t", buf)
	case Tstring:
		node.Content, buf = dprintf[string](info, "%s(%s): %s", buf)
	case Ttime:
		size, remain, err := checkDumpSizet(buf)
		if err != nil {
			return buf
		}

		rfc3339Str := string(remain[:size])
		t, err := time.Parse(time.RFC3339Nano, rfc3339Str)
		if err != nil {
			fmt.Println("parse time err:", err)
			return buf
		}
		node.Content = fmt.Sprintf("%s(%s): %s", info.name, info.rtype.String(), t.Format(time.RFC3339Nano))
		return remain[size:]
	case Tobject:
		sizet, fields, remain, err := checkDumpSize(buf)
		if err != nil {
			return buf
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: fields:%d)", info.name, info.rtype.String(), sizet, fields)
		for i := range fields {
			remain = info.info[i].parse(node, remain)
		}
		return remain
	case Tarray:
		sizet, arrayN, remain, err := checkDumpSize(buf)
		if err != nil {
			return buf
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)", info.name, info.rtype.String(), sizet, arrayN)
		for i := 0; i < arrayN; i++ {
			remain = info.info[0].parse(node, remain)
		}
		return remain
	case Tarraym: //[Tarraym][Ttype][sizet][sizea][data]
		sizet, arrayN, remain, err := checkDumpSize(buf[TSSD_TYPE_LENGTH:])
		if err != nil {
			return buf
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)%s[", info.name, info.rtype.String(), sizet, arrayN, info.info[0].rtype.String())
		for i := 0; i < arrayN; i++ {
			node.Content += parseMergeArray(&info.info[0], int8(buf[1]), remain)
			remain = remain[info.info[0].size:]
		}
		node.Content += "]"
		return remain
	case Tdict:
		sizet, mapLen, remain, err := checkDumpSize(buf)
		if err != nil {
			return buf
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)", info.name, info.rtype.String(), sizet, mapLen)
		fmt.Println("mapLen:", mapLen, node.Content, remain)
		for k := 0; k < mapLen; k++ {
			kvNode := &Node{
				Content: fmt.Sprintf("KVNode[%d]", k),
			}
			node.Children = append(node.Children, kvNode)
			remain = info.info[0].parse(kvNode, remain[1:])
			remain = info.info[1].parse(kvNode, remain[1:])
		}

		return remain
	default:
		fmt.Println("error: not support type:", buf[0])
	}
	return buf
}

func (info *typeInfo) print(data []byte) {

	fmt.Printf("==TType list: Tbool: %d, Tint64: %d, Tfloat64: %d, Tstring: %d, Tarray: %d, Tarraym: %d, Ttime: %d, Tenum: %d, Tobject: %d, Tdict: %d==\n",
		Tbool, Tint64, Tfloat64, Tstring, Tarray, Tarraym, Ttime, Tenum, Tobject, Tdict)

	root := &Node{
		Content: fmt.Sprintf("root-%s(%s)", info.name, info.rtype.String()),
	}
	info.parse(root, data)

	print.Print(root, (*Node).getChildren, (*Node).getContent, true)
}

// print your tssd []byte
func (factory factory) print(version string, data []byte) error {

	if _, ok := factory.versions[version]; !ok {
		return ErrorTSSDDataUnregister
	}

	info := factory.versions[version].info

	root := &Node{
		Content: "TSSD",
	}

	header, remain, err := dumpHeader(data)
	fmt.Println("after header:", remain, err, info)
	headerNode := &Node{
		Content: "header(header)",
		Children: []*Node{
			&Node{Content: fmt.Sprintf("Magic([4]byte):{%s}", string(header.Magic[:]))},
			&Node{Content: fmt.Sprintf("Version(int16):{%d}", header.Version)},
			&Node{Content: fmt.Sprintf("Schema:{%s %s %s}", header.Schema.Hash, header.Schema.Type, header.Schema.Content)},
		},
	}

	root.Children = append(root.Children, headerNode)
	info.parse(root, remain)

	print.Print(root, (*Node).getChildren, (*Node).getContent, true)
	return nil
}

func Print(flat Flatable, data []byte) error {
	factory, ok := groups[flat.Group()]
	if !ok {
		return ErrorTSSDDataUnregister
	}
	return factory.print(flat.Version(), data)
}
