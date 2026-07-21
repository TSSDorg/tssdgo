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

func dprintf[T comparable](info *typeInfo, format string, buf *Buffer) (string, error) {
	var d T
	err := info.dump(info, buf, Ptr(&d))
	return fmt.Sprintf(format, info.name, info.rtype.String(), d), err
}

func dprintf2[T comparable](info *typeInfo, format string, buf *Buffer) (string, error) {
	var d T
	_, err := buf.Read(Slice(Ptr(&d), Size_t(info.size)))
	return fmt.Sprintf(format, d), err
}

func parseMergeArray(info *typeInfo, ttype int8, buf *Buffer) (string, error) {
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
	return "", nil
}

func (info *typeInfo) parse(parent *Node, buf *Buffer) error {
	b, err := buf.PeekByte()
	if err != nil {
		return err
	}

	if info.Type != int8(b) && -info.Type != int8(b) {
		fmt.Println("print parsetype mismatch ", b, info.Type)
		return err
	}

	node := &Node{}
	parent.Children = append(parent.Children, node)
	switch int8(b) {
	case Tint8:
		node.Content, err = dprintf[int8](info, "%s(%s): %d", buf)
	case Tuint8:
		node.Content, err = dprintf[uint8](info, "%s(%s): %d", buf)
	case Tint16:
		node.Content, err = dprintf[int16](info, "%s(%s): %d", buf)
	case Tuint16:
		node.Content, err = dprintf[uint16](info, "%s(%s): %d", buf)
	case Tint32:
		node.Content, err = dprintf[int32](info, "%s(%s): %d", buf)
	case Tuint32:
		node.Content, err = dprintf[uint32](info, "%s(%s): %d", buf)
	case Tint64:
		node.Content, err = dprintf[int64](info, "%s(%s): %d", buf)
	case Tuint64:
		node.Content, err = dprintf[uint64](info, "%s(%s): %d", buf)
	case Tfloat32:
		node.Content, err = dprintf[float32](info, "%s(%s): %f", buf)
	case Tfloat64:
		node.Content, err = dprintf[float64](info, "%s(%s): %f", buf)
	case Tbool:
		node.Content, err = dprintf[bool](info, "%s(%s): %t", buf)
	case Tstring:
		node.Content, err = dprintf[string](info, "%s(%s): %s", buf)
	case Ttime:
		var t time.Time
		err := info.timeDump(buf, Ptr(&t))
		if err != nil {
			return err
		}
		node.Content = fmt.Sprintf("%s(%s): %s", info.name, info.rtype.String(), t.Format(time.RFC3339Nano))
		return nil
	case Tobject:
		_, err := buf.ReadByte()
		if err != nil {
			return err
		}
		sizet, fields, err := buf.checkDumpSize()
		if err != nil {
			return err
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: fields:%d)", info.name, info.rtype.String(), sizet, fields)
		for i := range fields {
			if err = info.info[i].parse(node, buf); err != nil {
				return err
			}
		}
		return nil
	case Tarray:
		_, err := buf.ReadByte()
		if err != nil {
			return err
		}
		sizet, arrayN, err := buf.checkDumpSize()
		if err != nil {
			return err
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)", info.name, info.rtype.String(), sizet, arrayN)
		for i := 0; i < arrayN; i++ {
			if err = info.info[0].parse(node, buf); err != nil {
				return err
			}
		}
		return nil
	case Tarraym: //[Tarraym][Ttype][sizet][sizea][data]
		b, err := buf.Read(make([]byte, 2))
		if err != nil {
			return err
		}
		sizet, arrayN, err := buf.checkDumpSize()
		if err != nil {
			return err
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)%s[", info.name, info.rtype.String(), sizet, arrayN, info.info[0].rtype.String())
		for i := 0; i < arrayN; i++ {
			s, err := parseMergeArray(&info.info[0], int8(b[1]), buf)
			if err != nil {
				return err
			}
			node.Content += s
		}
		node.Content += "]"
		return nil
	case Tdict:
		_, err := buf.ReadByte()
		if err != nil {
			return err
		}
		sizet, mapLen, err := buf.checkDumpSize()
		if err != nil {
			return err
		}

		node.Content = fmt.Sprintf("%s(%s[totalSize:%d]: len:%d)", info.name, info.rtype.String(), sizet, mapLen)
		fmt.Println("mapLen:", mapLen, node.Content)
		for k := 0; k < mapLen; k++ {
			kvNode := &Node{
				Content: fmt.Sprintf("KVNode[%d]", k),
			}
			node.Children = append(node.Children, kvNode)
			b, err = buf.ReadByte()
			if err != nil {
				return err
			}
			info.info[0].parse(kvNode, buf)
			b, err = buf.ReadByte()
			if err != nil {
				return err
			}
			info.info[1].parse(kvNode, buf)
		}
		return nil
	default:
		fmt.Println("error: not support type:", b)
	}
	return nil
}

func (info *typeInfo) print(buf Buffer) {

	fmt.Printf("==TType list: Tbool: %d, Tint64: %d, Tfloat64: %d, Tstring: %d, Tarray: %d, Tarraym: %d, Ttime: %d, Tenum: %d, Tobject: %d, Tdict: %d==\n",
		Tbool, Tint64, Tfloat64, Tstring, Tarray, Tarraym, Ttime, Tenum, Tobject, Tdict)

	root := &Node{
		Content: fmt.Sprintf("root-%s(%s)", info.name, info.rtype.String()),
	}
	info.parse(root, &buf)

	print.Print(root, (*Node).getChildren, (*Node).getContent, true)
}

// print your tssd []byte
func (factory factory) print(version string, buf *Buffer) error {
	if _, ok := factory.versions[version]; !ok {
		return ErrorTSSDDataSchemaUnmatch
	}

	info := factory.versions[version].info

	root := &Node{
		Content: "TSSD",
	}

	if len(buf.Fragments) == 0 {
		buf = Pipe(buf)
	}

	//header, err := //dumpHeader(buf)
	frag := &buf.Fragments[0]
	header := frag.Header
	fmt.Println("header:", header, info)
	headerNode := &Node{
		Content: "header(header)",
		Children: []*Node{
			&Node{Content: fmt.Sprintf("Magic([5]byte):{%s}", string(header.Magic[:]))},
			&Node{Content: fmt.Sprintf("Version[major.minor]:%d.%d", header.Version[1], header.Version[0])},
			&Node{Content: fmt.Sprintf("Schema:{%s %s %d %s}", frag.Schema.Hash, frag.Schema.TID, frag.Schema.Fragment, frag.Schema.Extent)},
		},
	}

	root.Children = append(root.Children, headerNode)
	info.parse(root, buf)

	print.Print(root, (*Node).getChildren, (*Node).getContent, true)
	return nil
}

func Print(flat Flatable, buf Buffer) error {
	factory, ok := groups[flat.Group()]
	if !ok {
		return ErrorTSSDDataSchemaUnmatch
	}
	return factory.print(flat.Version(), &buf)
}
