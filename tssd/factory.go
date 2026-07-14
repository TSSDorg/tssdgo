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

func (factory *factory) validate(header Header) error {
	if header.Version[1] != TSSD_VERSION_MAJOR || header.Version[0] != TSSD_VERSION_MINOR {
		return ErrorInvalidTSSDVersion
	}
	if _, ok := factory.schemas[header.Schema.Hash]; !ok {
		return ErrorTSSDDataSchemaUnmatch
	}
	return nil
}

func (factory *factory) marshalTo(flat Flatable, buf *Buffer) error {
	bi, ok := factory.versions[flat.Version()]
	if !ok {
		return ErrorTSSDDataSchemaUnmatch
	}

	appendHeader(buf, flat.Schema())
	return bi.info.marshalTo(flat, buf)
}

// UnmarshalTo direct unmarshal to your object
func (factory *factory) unmarshalTo(buf *Buffer, dest Flatable) error {
	header, err := dumpHeader(buf)
	if err != nil {
		return err
	}

	if err := factory.validate(*header); err != nil {
		return err
	}
	if err := dest.OnHeader(*header); err != nil {
		return err
	}
	remoteHash := header.Schema.Hash
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
	err = bi.info.unmarshalTo(buf, obj)
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
	header, err := dumpHeader(buf)
	if err != nil {
		return nil, err
	}

	if err := factory.validate(*header); err != nil {
		return nil, err
	}

	if err := factory.versions[factory.current].builder.OnHeader(*header); err != nil {
		return nil, err
	}
	remoteHash := header.Schema.Hash

	bi, ok := factory.schemas[remoteHash]
	if !ok {
		fmt.Printf("remote schema hash[%s] not found(unregisted)\n", remoteHash)
		return nil, ErrorTSSDDataSchemaUnmatch
	}
	obj := bi.builder.Build()

	err = bi.info.unmarshalTo(buf, obj)
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
