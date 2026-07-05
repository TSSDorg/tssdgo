package tssd

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
)

var groups = map[string]*factory{}

func Register(flat Flatable) {
	g := flat.Group()
	if _, ok := groups[g]; !ok {
		factory := &factory{
			versions: make(map[string]*buildInfo, 0),
			schemas:  make(map[string]*buildInfo, 0),
		}
		groups[g] = factory
		factory.register(flat)
		factory.current = flat.Version()
		return
	}
	groups[g].register(flat)
}

// default the first register one regard as current
// but we can let user overritten it by the new api
func RegisterCurrent(flat Flatable) {
	Register(flat)
	groups[flat.Group()].current = flat.Version()
}

// return current version of the register group
func CurrentVersion(group string) string {
	if factory, ok := groups[group]; ok {
		return factory.current
	}
	return ""
}

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

type Flatable interface {
	//to produce a flatable object
	Build() Flatable

	//digest algo, default is the md5sum, you can override it
	Hash([]byte) string

	//return the raw type []byte, you can call but should not override it
	//Types() []byte

	//schema write in the TSSD header
	//you can override it to add more infos such as a big json object string
	//but you need override OnHeader too
	Schema() Schema

	//when read/received a TSSD header, parse TSSD version and
	//parse schema and validate you received
	//return none-nil error will block factory to Unmarsh
	OnHeader(header Header) (err error)

	//return ver of the object, such as V1
	Version() string
	//return group of this class, suggest base class name, EG: Student
	Group() string

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

type Flat[T any, PT constrainFlatable[T]] struct {
}

func (this *Flat[T, PT]) Build() Flatable {
	return PT(new(T))
}

func (*Flat[T, PT]) Version() string {
	return TSSD_FLAT_KIND
}

func (*Flat[T, PT]) Group() string {
	return TSSD_FLAT_KIND
}

func (*Flat[T, PT]) Progeny() string {
	return ""
}

func (this *Flat[T, PT]) Types() []byte {
	obj := this.Build()
	g, version := obj.Group(), obj.Version()
	return groups[g].versions[version].info.types()
}

func (this *Flat[T, PT]) Hash(types []byte) string {
	hasher := md5.New()
	hasher.Write(types)                         // Write the data to the hasher
	hashBytes := hasher.Sum(nil)                // Get the hash sum as a byte slice
	hashString := hex.EncodeToString(hashBytes) // Convert to a hex string
	l := len(hashString)
	return hashString[:5] + hashString[l-5:l]
}

func (this *Flat[T, PT]) Schema() Schema {
	return Schema{
		this.Hash(this.Types()),
		"",
		"",
	}
}

// we need user can override it
func (this *Flat[T, PT]) OnHeader(header Header) error {
	return nil
}

func (this *Flat[T, PT]) Decorate(flat Flatable) Flatable {
	return flat
}

func (factory *factory) register(flat Flatable) {
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
	bi.schema = flat.Schema()
	hash := bi.schema.Hash
	bi.hash = hash
	factory.schemas[hash] = bi
}

func (factory *factory) Validate(header Header) error {
	if header.Version != TSSD_VERSION {
		return ErrorInvalidTSSDVersion
	}
	if _, ok := factory.schemas[header.Schema.Hash]; !ok {
		return ErrorTSSDDataSchemaReject
	}
	return nil
}

func MarshalTo(flat Flatable, to []byte) ([]byte, error) {
	if f, ok := groups[flat.Group()]; ok {
		return f.marshalTo(flat, to)
	}
	return nil, ErrorTSSDDataUnregister
}

func (factory *factory) marshalTo(flat Flatable, dest []byte) ([]byte, error) {
	bi, ok := factory.versions[flat.Version()]
	if !ok {
		return nil, ErrorTSSDDataSchemaReject
	}

	dest = appendHeader(dest, flat.Schema())
	return bi.info.marshal(flat, dest)
}

func Marshal(flat Flatable) ([]byte, error) {
	return MarshalTo(flat, make([]byte, 0, 4096))
}

func (factory *factory) marshal(flat Flatable) ([]byte, error) {
	return factory.marshalTo(flat, make([]byte, 0, 4096))
}

func UnmarshalTo(from []byte, to Flatable) (remain []byte, err error) {
	g := to.Group()
	if _, ok := groups[g]; !ok {
		return nil, ErrorTSSDDataUnregister
	}

	return groups[g].unmarshalTo(from, to)
}

func Unmarshal(from []byte, group string) (to Flatable, remain []byte, err error) {
	if f, ok := groups[group]; ok {
		return f.unmarshal(from)
	}

	return nil, nil, ErrorTSSDDataUnregister
}

// UnmarshalTo direct unmarshal to your object
func (factory *factory) unmarshalTo(src []byte, dest Flatable) ([]byte, error) {
	header, remain, err := dumpHeader(src)
	if err != nil {
		return src, err
	}

	if err := factory.Validate(*header); err != nil {
		return src, err
	}

	if err := dest.OnHeader(*header); err != nil {
		return src, err
	}
	remoteHash := header.Schema.Hash

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
func (factory *factory) decorate(flat, to Flatable) (Flatable, error) {
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
func (factory *factory) unmarshal(src []byte) (Flatable, []byte, error) {
	header, remain, err := dumpHeader(src)
	if err != nil {
		return nil, src, err
	}

	if err := factory.Validate(*header); err != nil {
		return nil, src, err
	}

	if err := factory.versions[factory.current].builder.OnHeader(*header); err != nil {
		return nil, src, err
	}
	remoteHash := header.Schema.Hash

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
