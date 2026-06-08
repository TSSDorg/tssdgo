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
	for i:=0; i<len(node.Children); i++ {
		node.Children[i].print()
	}
}

func dprintf[T comparable](info *typeInfo, format string, buf []byte) (string, []byte) {
	var d T
	remain, _ := info.dump(info, buf, Ptr(&d))
	return fmt.Sprintf(format, info.Name, info.rtype.String(), d), remain
}

func (info *typeInfo) parse(parent *Node, buf []byte) []byte {
	if info.Type != int8(buf[0]) && -info.Type != int8(buf[0]) {
		fmt.Println("type mismatch ", info.Type, buf[0])
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
		fmt.Println("node Content string:", node.Content)
	case Ttime:
		if len(buf) < 3 {
			fmt.Println("need more data size: ", len(buf))
			return buf
		}
		size := int(dumpSize(buf[1:]))
		if len(buf) < 3+size {
			fmt.Println("need more data time: ", len(buf))
			return buf
		}
		rfc3339Str := string(buf[3 : 3+size])
		t, err := time.Parse(time.RFC3339Nano, rfc3339Str)
		if err != nil {
			fmt.Println("parse time err:", err)
			return buf
		}
		node.Content = fmt.Sprintf("%s(%s): %s", info.Name, info.rtype.String(), t.Format(time.RFC3339Nano))

		return buf[3+size:]

	case Tobject:
		if len(buf) < 5 {
			fmt.Println("need more data size: ", len(buf))
			return buf
		}
		size := int(dumpSize(buf[1:]))
		if len(buf) < 3+size {
			fmt.Println("need more data size: ", len(buf), size)
			return buf
		}
		fields := int(dumpSize(buf[3:]))

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: fields:%d)", info.Name, info.rtype.String(), size, fields)
		remain := buf[5:]
		for i := range fields {
			remain = info.info[i].parse(node, remain)
		}
		return remain
	case Tarray:
		if len(buf) < 5 {
			fmt.Println("need more data: ", len(buf))
			return buf
		}
		size := int(dumpSize(buf[1:]))
		if len(buf) < 3+size {
			fmt.Println("need more data size: ", len(buf), size)
			return buf
		}
		arrayN := int(dumpSize(buf[3:]))

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)", info.Name, info.rtype.String(), size, arrayN)
		remain := buf[5:]
		for i := 0; i < arrayN; i++ {
			remain = info.info[0].parse(node, remain)
		}
		return remain
	case Tdict:
		if len(buf) < 3 {
			fmt.Println("need more data for size: ", len(buf))
			return buf
		}
		size := int(dumpSize(buf[1:]))
		if len(buf) < 3+size {
			fmt.Println("need more data for the map: ", len(buf))
			return buf
		}

		mapLen := int(dumpSize(buf[3:]))
		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)", info.Name, info.rtype.String(), size, mapLen)
		remain := buf[5:]
		fmt.Println("mapLen:", mapLen, node.Content, remain)
		for k := 0; k < mapLen; k++ {
			kvNode := &Node {
				Content: fmt.Sprintf("KVNode[%d]", k),
			}
			node.Children = append(node.Children, kvNode)
			remain = info.info[0].parse(kvNode, remain)
			remain = info.info[1].parse(kvNode, remain)
		}

		return remain
	default:
		fmt.Println("error: not support type:", buf[0])
	}
	return buf
}

func (info *typeInfo) print(data []byte) {
	
	root := &Node{
		Content: fmt.Sprintf("root-%s(%s)", info.Name, info.rtype.String()),
	}
	info.parse(root, data)
	
	print.Print(root, (*Node).getChildren, (*Node).getContent)
}

//print your tssd []byte
func (factory Factory) Print(version string, data []byte) {
	//factory.versions[version].info.print(data)

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
			&Node{Content: fmt.Sprintf("Schema:{%s}", header.Schema)},
		},
	}

	root.Children = append(root.Children, headerNode)
	info.parse(root, remain)
	
	print.Print(root, (*Node).getChildren, (*Node).getContent)

}
