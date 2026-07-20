package tssd

import (
	"fmt"
	"reflect"
)

type buildInfo struct {
	version string //current version
	progeny string //which version it can upgrade after decoration
	schema  Schema //current schema
	hash    string
	info    *typeInfo
	builder Flatable //keep it as builder
}

type factory struct {
	current  string
	versions map[string]*buildInfo //local we seek by names or version
	schemas  map[string]*buildInfo //and remote we seek by schema
}

func (factory *factory) register(flat Flatable) {
	if _, ok := factory.versions[flat.Version()]; ok {
		//skip repeat register
		return
	}

	value := reflect.ValueOf(flat)
	v := value.Type().Elem()

	info := parse(reflect.New(v).Elem().Interface())
	bi := &buildInfo{
		version: flat.Version(),
		progeny: flat.Progeny(),
		info:    info,
		builder: flat.Build(), //we build a new one, rather user's data
	}

	factory.versions[flat.Version()] = bi
	bi.schema = flat.Schema()
	hash := bi.schema.Hash
	bi.hash = hash
	factory.schemas[hash] = bi
}

func (factory *factory) marshalTo(flat Flatable, buf *Buffer) error {
	bi, ok := factory.versions[flat.Version()]
	if !ok {
		return ErrorTSSDDataSchemaUnmatch
	}

	if err := buf.prepare(flat.Schema()); err != nil {
		return err
	}
	err := bi.info.marshalTo(flat, buf)
	buf.finish()
	return err
}

// UnmarshalTo direct unmarshal to your object
func (factory *factory) unmarshalTo(buf *Buffer, dest Flatable) error {
	if len(buf.Fragments) == 0 {
		return ErrorInSufficientData
	}
	remoteHash := buf.Fragments[0].Schema.Hash
	local := factory.versions[dest.Version()].hash
	bi, ok := factory.schemas[remoteHash]
	if !ok {
		fmt.Printf("remote schema hash[%s] not found(unregisted), local:[%s]\n", remoteHash, local)
		return ErrorTSSDDataSchemaUnmatch
	}

	if local == remoteHash {
		return bi.info.unmarshalTo(buf, dest)
	}

	obj := bi.builder.Build()
	err := bi.info.unmarshalTo(buf, obj)
	if err != nil {
		return err
	}

	_, err = factory.decorate(obj, dest)
	return err
}

// chain upgate it to the latest
func (factory *factory) decorate(flat, to Flatable) (Flatable, error) {
	for v := flat.Progeny(); len(v) > 0; {
		if v == to.Version() {
			return to.Decorate(flat), nil
		}
		bi, ok := factory.versions[v]
		if !ok {
			fmt.Printf("local version %s not found\n", v)
			return nil, ErrorTSSDDataSchemaUnmatch
		}
		flat = bi.builder.Build().Decorate(flat)
		v = flat.Progeny()
	}
	//your may specify unmarshal to a old one
	return nil, ErrorTSSDDataSchemaUnmatch
}

// Unmarshal we new a current version object for user
func (factory *factory) unmarshal(buf *Buffer) (Flatable, error) {
	if len(buf.Fragments) == 0 {
		return nil, ErrorInSufficientData
	}
	remoteHash := buf.Fragments[0].Schema.Hash

	bi, ok := factory.schemas[remoteHash]
	if !ok {
		fmt.Printf("remote schema hash[%s] not found(unregisted)\n", remoteHash)
		return nil, ErrorTSSDDataSchemaUnmatch
	}
	obj := bi.builder.Build()

	err := bi.info.unmarshalTo(buf, obj)
	if err != nil {
		return nil, err
	}

	v := obj.Progeny()
	if len(v) == 0 || obj.Version() == factory.current {
		return obj, nil
	}

	to := factory.versions[factory.current].builder.Build()
	return factory.decorate(obj, to)
}
