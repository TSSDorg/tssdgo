package tssd

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
)

type BuildFunc = func() Flatable

type buildInfo struct {
	version string //current version
	progeny string //which version it can upgrade after decoration
	schema  string //current schema
	hash    string
	info    *typeInfo
	builder Flatable //keep it as builder
}

type Factory struct {
	current  string
	versions map[string]*buildInfo //local we seek by names or version
	schemas  map[string]*buildInfo //and remote we seek by schema
}

type Flatable interface {
	//to produce a flatable object
	Build() Flatable

	//digest algo, default is the md5sum, you can override it
	Hash([]byte) string

	//return the raw type []byte, you can call but should not override it
	Types(Factory) []byte

	//schema write in the TSSD header
	//you can override it to add more infos such as a big json object string
	//but you need override OnSchema too
	Schema(Factory) string

	//parse schema you received and return hash
	//Or process and do other checks when you write other info in schema
	//return none-nil error will block factory to Unmarsh
	OnSchema(factory Factory, schema string) (hash string, err error)

	//the current version or class name of the object
	Version() string

	//Progeny or Successor of current version
	//which version it can upgrade to after Decorate
	//default it should return "", which means latest
	Progeny() string

	//After Unmarshal, Decorate the object to support convert some info or migration/upgrate the object
	Decorate(Flatable) Flatable
}

type constrainFlatable[T any] interface {
	Flatable
	*T
}

type Flat[T any, PT constrainFlatable[T]] struct{}

func (*Flat[T, PT]) Build() Flatable {
	return PT(new(T))
}

func (*Flat[T, PT]) Version() string {
	return TSSD_FLAT_KIND
}

func (*Flat[T, PT]) Progeny() string {
	return ""
}

func (this *Flat[T, PT]) Types(factory Factory) []byte {
	version := this.Build().Version()
	return factory.versions[version].info.types()
}

func (this *Flat[T, PT]) Hash(types []byte) string {
	hasher := md5.New()
	hasher.Write(types)                         // Write the data to the hasher
	hashBytes := hasher.Sum(nil)                // Get the hash sum as a byte slice
	hashString := hex.EncodeToString(hashBytes) // Convert to a hex string
	l := len(hashString)
	return hashString[:5] + hashString[l-5:l]
}

func (this *Flat[T, PT]) Schema(factory Factory) string {
	return this.Hash(this.Types(factory))
}

// we need let user can override it
func (this *Flat[T, PT]) OnSchema(factory Factory, schema string) (hash string, err error) {
	return schema, nil
}

func (this *Flat[T, PT]) Decorate(flat Flatable) Flatable {
	return flat
}

func New(flat Flatable) Factory {
	factory := Factory{
		versions: make(map[string]*buildInfo, 0),
		schemas:  make(map[string]*buildInfo, 0),
	}
	factory.Register(flat)
	factory.current = flat.Version()
	return factory
}

func (factory Factory) Register(flat Flatable) {
	if factory.current == flat.Version() {
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
	bi.schema = flat.Schema(factory)
	hash, _ := flat.OnSchema(factory, bi.schema)
	bi.hash = hash
	factory.schemas[hash] = bi
}

func (factory Factory) MarshalTo(flat Flatable, dest []byte) ([]byte, error) {
	bi, ok := factory.versions[flat.Version()]
	if !ok {
		return nil, ErrorTSSDDataSchemaReject
	}

	dest = appendHeader(dest, factory.versions[flat.Version()].schema)
	return bi.info.marshal(flat, dest)
}

func (factory Factory) Marshal(flat Flatable) ([]byte, error) {

	//TODO: maybe we should mashal current version obj only ?
	if _, ok := factory.versions[flat.Version()]; ok {
		dest := make([]byte, 0, 4096)
		return factory.MarshalTo(flat, dest)
	}

	return nil, ErrorTSSDDataSchemaReject
}

// UnmarshalTo direct unmarshal to your object
func (factory Factory) UnmarshalTo(src []byte, dest Flatable) ([]byte, error) {
	header, remain, err := dumpHeader(src)
	if err != nil {
		return src, err
	}

	remoteHash, err := dest.OnSchema(factory, header.Schema)
	if err != nil {
		return src, err
	}

	//local := dest.Schema(factory)   //default it new objct, so we fetch it
	local := factory.versions[dest.Version()].hash
	bi, ok := factory.schemas[remoteHash]
	if !ok {
		fmt.Printf("local schema: %s doesn't match with remote: schema[%s] hash[%s]\n", local, header.Schema, remoteHash)
		return src, ErrorTSSDDataSchemaReject
	}

	if local == remoteHash {
		return bi.info.unmarshalTo(remain, dest)
	}

	obj := bi.builder.Build()
	remain, err = bi.info.unmarshalTo(remain, obj)
	if err != nil {
		return src, err
	}

	_, err = factory.decorate(obj, dest)
	if err != nil {
		return src, err
	}

	return remain, nil
}

// chain upgate it to the latest
func (factory Factory) decorate(flat, to Flatable) (Flatable, error) {
	for v := flat.Progeny(); len(v) > 0; {
		if v == to.Version() {
			return to.Decorate(flat), nil
		}
		bi, ok := factory.versions[v]
		if !ok {
			fmt.Printf("local version %s not found\n", v)
			return nil, ErrorTSSDDataSchemaReject
		}
		flat = bi.builder.Build().Decorate(flat)
		v = flat.Progeny()
	}
	//your may specify unmarshal to a old one
	return nil, ErrorTSSDDataSchemaReject
}

// Unmarshal we new a current version object for user and return the remain bytes after consum
func (factory Factory) Unmarshal(src []byte) (Flatable, []byte, error) {
	header, remain, err := dumpHeader(src)
	if err != nil {
		return nil, src, err
	}

	remoteHash, err := factory.versions[factory.current].builder.OnSchema(factory, header.Schema)
	if err != nil {
		return nil, src, err
	}

	bi, ok := factory.schemas[remoteHash]
	if !ok {
		fmt.Printf("remote schema [%s] hash[%s] not found\n", header.Schema, remoteHash)
		return nil, src, ErrorTSSDDataSchemaReject
	}
	obj := bi.builder.Build()

	remain, err = bi.info.unmarshalTo(remain, obj)
	if err != nil {
		return nil, src, err
	}

	v := obj.Progeny()
	if len(v) == 0 || obj.Version() == factory.current {
		return obj, remain, nil
	}

	to := factory.versions[factory.current].builder.Build()

	flat, err := factory.decorate(obj, to)
	if err != nil {
		return nil, src, err
	}

	return flat, remain, nil
}
