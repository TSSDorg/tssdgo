package tssd

import (
	"fmt"
	"math/rand"
)

var registers struct {
	families map[string]*factory
	types    map[string] struct {   //convert remote Types to local flat info
		family  string
		version string
	}
}

func init() {
	registers.families = make(map[string]*factory)
	registers.types = make(map[string]struct {
		family string
		version string
	})
}


func Register(flat Flatable) {
	family := flat.Family()
	_, ok := registers.families[family]
	if !ok {
		registers.families[family] = &factory{
			current:  flat.Version(), //register first one as the current
			versions: make(map[string]*buildInfo, 0),
		}
	}
	registers.families[family].register(flat)
	schema :=  flat.Schema()
	fmt.Println("~~~~Register schema:", schema)
	registers.types[schema.Types] = struct {
		family  string
		version string
	} {
		flat.Family(),
		flat.Version(),
	}
}

func getBuildInfoByTypes(types string) (*buildInfo, error) {
	fv, ok := registers.types[types]
	if !ok {
		return nil, ErrorTSSDDataSchemaUnmatch
	}
	return registers.families[fv.family].versions[fv.version], nil
}


// default the first register one regard as current
// but we can let user overritten it by the new api
func RegisterCurrent(flat Flatable) {
	Register(flat)
	registers.families[flat.Family()].current = flat.Version()
}

// return current version of the register family
// return "" if family not exist
func CurrentVersion(family string) string {
	if factory, ok := registers.families[family]; ok {
		return factory.current
	}
	return ""
}

type Flatable interface {
	//to produce a flatable object
	Build() Flatable

	//schema write in the TSSD header
	//but you need override OnHeader to receive it
	Schema() Schema

	//when read/received a TSSD header, parse TSSD version and
	//parse schema and validate you received
	//return none-nil error will block factory to Unmarsh
	//OnHeader(header Header) (err error)

	//return family of this class, suggest base class name, EG: Student
	Family() string

	//return ver of the object, such as V1
	Version() string

	//Progeny or Successor of current version
	//which version it can upgrade to after Decorate
	//default it should return "", which means latest
	Progeny() string

	//After Unmarshal, Decorate the object to support convert some info or migration/upgrate the object
	Decorate(Flatable) Flatable

	//id for current object, save into schema
	TID() string

	//extent info in schema
	Info() string
}

type constrainFlatable[T any] interface {
	Flatable
	*T
}

type Flat[T any, PT constrainFlatable[T]] struct{}

func (this *Flat[T, PT]) Build() Flatable {
	return PT(new(T))
}

func (*Flat[T, PT]) Version() string {
	return TSSD_FLAT_KIND
}

func (*Flat[T, PT]) Family() string {
	return TSSD_FLAT_KIND
}

func (*Flat[T, PT]) Progeny() string {
	return ""
}

func (this *Flat[T, PT]) Types() []byte {
	obj := this.Build()
	g, version := obj.Family(), obj.Version()
	fmt.Println(g, version, " Types:", registers.families[g].versions[version].info.types())
	return registers.families[g].versions[version].info.types()
}


func (this *Flat[T, PT]) Schema() Schema {
	fmt.Println("Types:", this.Types(), ", hash:", string(HashFunc(this.Types())))
	return Schema{
		-1,
		this.TID(),
		string(HashFunc(this.Types())),
		this.Info(),
	}
}

func (this *Flat[T, PT]) TID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)

	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func (*Flat[T, PT]) Info() string {
	return ""
}

func (this *Flat[T, PT]) Decorate(flat Flatable) Flatable {
	return this
}

func Marshal(flat Flatable) (*Buffer, error) {
	buf := new(Buffer)
	return buf, MarshalTo(flat, buf)
}

func MarshalTo(flat Flatable, buf *Buffer) error {
	if factory, ok := registers.families[flat.Family()]; ok {
		return factory.marshalTo(flat, buf)
	}
	return ErrorTSSDDataSchemaUnmatch
}

func UnmarshalTo(buf *Buffer, to Flatable) error {
	if factory, ok := registers.families[to.Family()]; ok {
		return factory.unmarshalTo(buf, to)
	}
	return ErrorTSSDDataSchemaUnmatch
}

func Unmarshal(buf *Buffer, family string) (to Flatable, err error) {
	if factory, ok := registers.families[family]; ok {
		return factory.unmarshal(buf)
	}
	return nil, ErrorTSSDDataSchemaUnmatch
}
