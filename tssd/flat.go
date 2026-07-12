package tssd

import (
	"crypto/md5"
	"encoding/hex"
)

var groups = map[string]*factory{}

func Register(flat Flatable) {
	group := flat.Group()
	_, ok := groups[group]
	if !ok {
		groups[group] = &factory{
			current:  flat.Version(), //register first one as the current
			versions: make(map[string]*buildInfo, 0),
			schemas:  make(map[string]*buildInfo, 0),
		}
	}
	groups[group].register(flat)
}

// default the first register one regard as current
// but we can let user overritten it by the new api
func RegisterCurrent(flat Flatable) {
	Register(flat)
	groups[flat.Group()].current = flat.Version()
}

// return current version of the register group
// return "" if group not exist
func CurrentVersion(group string) string {
	if factory, ok := groups[group]; ok {
		return factory.current
	}
	return ""
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

type Flat[T any, PT constrainFlatable[T]] struct{}

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
	return this
}

func Marshal(flat Flatable) (*Buffer, error) {
	buf := new(Buffer)
	return buf, MarshalTo(flat, buf)
}

func MarshalTo(flat Flatable, buf *Buffer) error {
	if factory, ok := groups[flat.Group()]; ok {
		return factory.marshalTo(flat, buf)
	}
	return ErrorTSSDDataUnregister
}

func UnmarshalTo(from []byte, to Flatable) (remain []byte, err error) {
	if factory, ok := groups[to.Group()]; ok {
		return factory.unmarshalTo(from, to)
	}
	return nil, ErrorTSSDDataUnregister
}

func Unmarshal(from []byte, group string) (to Flatable, remain []byte, err error) {
	if factory, ok := groups[group]; ok {
		return factory.unmarshal(from)
	}
	return nil, nil, ErrorTSSDDataUnregister
}
