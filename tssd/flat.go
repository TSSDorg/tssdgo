package tssd

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
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

	//schema write in the TSSD header
	//but you need override OnHeader to receive it
	Schema() Schema

	//when read/received a TSSD header, parse TSSD version and
	//parse schema and validate you received
	//return none-nil error will block factory to Unmarsh
	OnHeader(header Header) (err error)

	//return group of this class, suggest base class name, EG: Student
	Group() string

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
	Extent() string
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
		this.TID(),
		-1,
		this.Extent(),
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

func (*Flat[T, PT]) Extent() string {
	return ""
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
	return ErrorTSSDDataSchemaUnmatch
}

func UnmarshalTo(buf *Buffer, to Flatable) error {
	if factory, ok := groups[to.Group()]; ok {
		return factory.unmarshalTo(buf, to)
	}
	return ErrorTSSDDataSchemaUnmatch
}

func Unmarshal(buf *Buffer, group string) (to Flatable, err error) {
	if factory, ok := groups[group]; ok {
		return factory.unmarshal(buf)
	}
	return nil, ErrorTSSDDataSchemaUnmatch
}
