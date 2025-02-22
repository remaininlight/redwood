// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: blob.proto

package pb

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
	redwood_dev_types "redwood.dev/types"
	pb "redwood.dev/types/pb"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type ID struct {
	HashAlg pb.HashAlg             `protobuf:"varint,1,opt,name=hashAlg,proto3,enum=Redwood.types.HashAlg" json:"hashAlg,omitempty" tree:"hashAlg"`
	Hash    redwood_dev_types.Hash `protobuf:"bytes,2,opt,name=hash,proto3,customtype=redwood.dev/types.Hash" json:"hash" tree:"hash"`
}

func (m *ID) Reset()      { *m = ID{} }
func (*ID) ProtoMessage() {}
func (*ID) Descriptor() ([]byte, []int) {
	return fileDescriptor_6903d1e8a20272e8, []int{0}
}
func (m *ID) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ID.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ID.Merge(m, src)
}
func (m *ID) XXX_Size() int {
	return m.Size()
}
func (m *ID) XXX_DiscardUnknown() {
	xxx_messageInfo_ID.DiscardUnknown(m)
}

var xxx_messageInfo_ID proto.InternalMessageInfo

func (m *ID) GetHashAlg() pb.HashAlg {
	if m != nil {
		return m.HashAlg
	}
	return pb.HashAlgUnknown
}

type Manifest struct {
	TotalSize uint64          `protobuf:"varint,1,opt,name=totalSize,proto3" json:"totalSize,omitempty" tree:"totalSize"`
	Chunks    []ManifestChunk `protobuf:"bytes,2,rep,name=chunks,proto3" json:"chunks" tree:"chunks"`
}

func (m *Manifest) Reset()      { *m = Manifest{} }
func (*Manifest) ProtoMessage() {}
func (*Manifest) Descriptor() ([]byte, []int) {
	return fileDescriptor_6903d1e8a20272e8, []int{1}
}
func (m *Manifest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Manifest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Manifest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Manifest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Manifest.Merge(m, src)
}
func (m *Manifest) XXX_Size() int {
	return m.Size()
}
func (m *Manifest) XXX_DiscardUnknown() {
	xxx_messageInfo_Manifest.DiscardUnknown(m)
}

var xxx_messageInfo_Manifest proto.InternalMessageInfo

func (m *Manifest) GetTotalSize() uint64 {
	if m != nil {
		return m.TotalSize
	}
	return 0
}

func (m *Manifest) GetChunks() []ManifestChunk {
	if m != nil {
		return m.Chunks
	}
	return nil
}

type ManifestChunk struct {
	SHA3  redwood_dev_types.Hash `protobuf:"bytes,1,opt,name=sha3,proto3,customtype=redwood.dev/types.Hash" json:"sha3" tree:"sha3"`
	Range pb.Range               `protobuf:"bytes,2,opt,name=range,proto3" json:"range" tree:"range"`
}

func (m *ManifestChunk) Reset()      { *m = ManifestChunk{} }
func (*ManifestChunk) ProtoMessage() {}
func (*ManifestChunk) Descriptor() ([]byte, []int) {
	return fileDescriptor_6903d1e8a20272e8, []int{2}
}
func (m *ManifestChunk) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ManifestChunk) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ManifestChunk.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ManifestChunk) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ManifestChunk.Merge(m, src)
}
func (m *ManifestChunk) XXX_Size() int {
	return m.Size()
}
func (m *ManifestChunk) XXX_DiscardUnknown() {
	xxx_messageInfo_ManifestChunk.DiscardUnknown(m)
}

var xxx_messageInfo_ManifestChunk proto.InternalMessageInfo

func (m *ManifestChunk) GetRange() pb.Range {
	if m != nil {
		return m.Range
	}
	return pb.Range{}
}

func init() {
	proto.RegisterType((*ID)(nil), "Redwood.blob.ID")
	proto.RegisterType((*Manifest)(nil), "Redwood.blob.Manifest")
	proto.RegisterType((*ManifestChunk)(nil), "Redwood.blob.ManifestChunk")
}

func init() { proto.RegisterFile("blob.proto", fileDescriptor_6903d1e8a20272e8) }

var fileDescriptor_6903d1e8a20272e8 = []byte{
	// 435 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x92, 0xb1, 0x6f, 0xd3, 0x40,
	0x14, 0xc6, 0xef, 0x05, 0x53, 0xe0, 0x92, 0x56, 0xe8, 0x08, 0x55, 0x54, 0xa4, 0xe7, 0xe8, 0xa6,
	0x2c, 0xb5, 0xa5, 0x44, 0x2c, 0x9d, 0x88, 0xe9, 0xd0, 0x22, 0xb1, 0x5c, 0x27, 0xd8, 0xec, 0xe6,
	0x6a, 0x47, 0x84, 0x5c, 0x14, 0x3b, 0x20, 0x98, 0x3a, 0x33, 0x21, 0xb1, 0xf2, 0x07, 0xf0, 0x27,
	0x30, 0x32, 0x76, 0xcc, 0x58, 0x21, 0x64, 0xd5, 0x97, 0x85, 0xb1, 0xca, 0xc4, 0x88, 0x7c, 0xe7,
	0xe0, 0xc2, 0xc0, 0xf6, 0xee, 0xde, 0xf7, 0xfb, 0xfc, 0xdd, 0x27, 0x53, 0x1a, 0x4d, 0x54, 0xe4,
	0xcd, 0xe6, 0x2a, 0x53, 0xac, 0x25, 0xe4, 0xe8, 0xad, 0x52, 0x23, 0xaf, 0xbc, 0xdb, 0xdb, 0x8f,
	0xc7, 0x59, 0xb2, 0x88, 0xbc, 0x53, 0xf5, 0xda, 0x8f, 0x55, 0xac, 0x7c, 0x23, 0x8a, 0x16, 0x67,
	0xe6, 0x64, 0x0e, 0x66, 0xb2, 0xf0, 0x5e, 0x3b, 0x7b, 0x37, 0x93, 0xa9, 0x3f, 0x8b, 0x7c, 0x33,
	0xd8, 0x5b, 0xfe, 0x09, 0x68, 0xe3, 0xf8, 0x90, 0x1d, 0xd2, 0x3b, 0x49, 0x98, 0x26, 0xc3, 0x49,
	0xdc, 0x81, 0x2e, 0xf4, 0x76, 0xfa, 0xbb, 0xde, 0xe6, 0x5b, 0x56, 0x7d, 0x64, 0xb7, 0x01, 0x5b,
	0xe7, 0xee, 0x4e, 0x36, 0x97, 0xf2, 0x80, 0x57, 0x00, 0x17, 0x1b, 0x94, 0x0d, 0xa9, 0x53, 0x8e,
	0x9d, 0x46, 0x17, 0x7a, 0xad, 0x60, 0xff, 0x22, 0x77, 0xc9, 0xf7, 0xdc, 0xdd, 0x9d, 0x57, 0x4e,
	0x23, 0xf9, 0xc6, 0xaf, 0xdd, 0xd6, 0xb9, 0xdb, 0xac, 0x8d, 0xb8, 0x30, 0xe8, 0x81, 0x73, 0xfe,
	0xa3, 0x4b, 0xf8, 0x07, 0xa0, 0x77, 0x9f, 0x87, 0xd3, 0xf1, 0x99, 0x4c, 0x33, 0xd6, 0xa7, 0xf7,
	0x32, 0x95, 0x85, 0x93, 0x93, 0xf1, 0x7b, 0x69, 0xd2, 0x39, 0x41, 0x7b, 0x9d, 0xbb, 0xf7, 0x2d,
	0xfc, 0x67, 0xc5, 0x45, 0x2d, 0x63, 0xcf, 0xe8, 0xd6, 0x69, 0xb2, 0x98, 0xbe, 0x4a, 0x3b, 0x8d,
	0xee, 0xad, 0x5e, 0xb3, 0xff, 0xc8, 0xbb, 0x59, 0x9d, 0xb7, 0xf1, 0x7e, 0x5a, 0x6a, 0x82, 0x87,
	0x65, 0xd0, 0x75, 0xee, 0x6e, 0x5b, 0x47, 0x0b, 0x72, 0x51, 0x39, 0xf0, 0xcf, 0x40, 0xb7, 0xff,
	0x02, 0xd8, 0x31, 0x75, 0xd2, 0x24, 0x1c, 0x98, 0x30, 0xad, 0xe0, 0xf1, 0xff, 0xdf, 0xa9, 0x73,
	0xd7, 0x39, 0x39, 0x1a, 0x0e, 0xea, 0xf7, 0x96, 0x2c, 0x17, 0xc6, 0x82, 0x3d, 0xa1, 0xb7, 0xe7,
	0xe1, 0x34, 0x96, 0xa6, 0xb3, 0x66, 0xbf, 0xfd, 0x4f, 0xed, 0xa2, 0xdc, 0x05, 0xed, 0x2a, 0x60,
	0xcb, 0xf2, 0x06, 0xe0, 0xc2, 0x82, 0xc1, 0x8b, 0x65, 0x81, 0xe4, 0xb2, 0x40, 0x72, 0x55, 0x20,
	0x5c, 0x17, 0x08, 0xbf, 0x0a, 0x84, 0x73, 0x8d, 0xf0, 0x45, 0x23, 0x7c, 0xd5, 0x48, 0xbe, 0x69,
	0x84, 0x0b, 0x8d, 0xb0, 0xd4, 0x08, 0x57, 0x1a, 0xe1, 0xa7, 0x46, 0x72, 0xad, 0x11, 0x3e, 0xae,
	0x90, 0x2c, 0x57, 0x48, 0x2e, 0x57, 0x48, 0x5e, 0x3e, 0xb8, 0x19, 0xbe, 0xec, 0xc8, 0x9f, 0x45,
	0xd1, 0x96, 0xf9, 0x47, 0x06, 0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0x8e, 0x05, 0xa8, 0xac, 0x84,
	0x02, 0x00, 0x00,
}

func (this *ID) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*ID)
	if !ok {
		that2, ok := that.(ID)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *ID")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *ID but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *ID but is not nil && this == nil")
	}
	if this.HashAlg != that1.HashAlg {
		return fmt.Errorf("HashAlg this(%v) Not Equal that(%v)", this.HashAlg, that1.HashAlg)
	}
	if !this.Hash.Equal(that1.Hash) {
		return fmt.Errorf("Hash this(%v) Not Equal that(%v)", this.Hash, that1.Hash)
	}
	return nil
}
func (this *ID) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*ID)
	if !ok {
		that2, ok := that.(ID)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.HashAlg != that1.HashAlg {
		return false
	}
	if !this.Hash.Equal(that1.Hash) {
		return false
	}
	return true
}
func (this *Manifest) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*Manifest)
	if !ok {
		that2, ok := that.(Manifest)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *Manifest")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *Manifest but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *Manifest but is not nil && this == nil")
	}
	if this.TotalSize != that1.TotalSize {
		return fmt.Errorf("TotalSize this(%v) Not Equal that(%v)", this.TotalSize, that1.TotalSize)
	}
	if len(this.Chunks) != len(that1.Chunks) {
		return fmt.Errorf("Chunks this(%v) Not Equal that(%v)", len(this.Chunks), len(that1.Chunks))
	}
	for i := range this.Chunks {
		if !this.Chunks[i].Equal(&that1.Chunks[i]) {
			return fmt.Errorf("Chunks this[%v](%v) Not Equal that[%v](%v)", i, this.Chunks[i], i, that1.Chunks[i])
		}
	}
	return nil
}
func (this *Manifest) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Manifest)
	if !ok {
		that2, ok := that.(Manifest)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.TotalSize != that1.TotalSize {
		return false
	}
	if len(this.Chunks) != len(that1.Chunks) {
		return false
	}
	for i := range this.Chunks {
		if !this.Chunks[i].Equal(&that1.Chunks[i]) {
			return false
		}
	}
	return true
}
func (this *ManifestChunk) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*ManifestChunk)
	if !ok {
		that2, ok := that.(ManifestChunk)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *ManifestChunk")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *ManifestChunk but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *ManifestChunk but is not nil && this == nil")
	}
	if !this.SHA3.Equal(that1.SHA3) {
		return fmt.Errorf("SHA3 this(%v) Not Equal that(%v)", this.SHA3, that1.SHA3)
	}
	if !this.Range.Equal(&that1.Range) {
		return fmt.Errorf("Range this(%v) Not Equal that(%v)", this.Range, that1.Range)
	}
	return nil
}
func (this *ManifestChunk) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*ManifestChunk)
	if !ok {
		that2, ok := that.(ManifestChunk)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !this.SHA3.Equal(that1.SHA3) {
		return false
	}
	if !this.Range.Equal(&that1.Range) {
		return false
	}
	return true
}
func (this *ID) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&pb.ID{")
	s = append(s, "HashAlg: "+fmt.Sprintf("%#v", this.HashAlg)+",\n")
	s = append(s, "Hash: "+fmt.Sprintf("%#v", this.Hash)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *Manifest) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&pb.Manifest{")
	s = append(s, "TotalSize: "+fmt.Sprintf("%#v", this.TotalSize)+",\n")
	if this.Chunks != nil {
		vs := make([]ManifestChunk, len(this.Chunks))
		for i := range vs {
			vs[i] = this.Chunks[i]
		}
		s = append(s, "Chunks: "+fmt.Sprintf("%#v", vs)+",\n")
	}
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *ManifestChunk) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&pb.ManifestChunk{")
	s = append(s, "SHA3: "+fmt.Sprintf("%#v", this.SHA3)+",\n")
	s = append(s, "Range: "+strings.Replace(this.Range.GoString(), `&`, ``, 1)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringBlob(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *ID) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ID) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ID) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Hash.Size()
		i -= size
		if _, err := m.Hash.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintBlob(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if m.HashAlg != 0 {
		i = encodeVarintBlob(dAtA, i, uint64(m.HashAlg))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Manifest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Manifest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Manifest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Chunks) > 0 {
		for iNdEx := len(m.Chunks) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Chunks[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintBlob(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if m.TotalSize != 0 {
		i = encodeVarintBlob(dAtA, i, uint64(m.TotalSize))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ManifestChunk) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ManifestChunk) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ManifestChunk) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Range.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintBlob(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size := m.SHA3.Size()
		i -= size
		if _, err := m.SHA3.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintBlob(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintBlob(dAtA []byte, offset int, v uint64) int {
	offset -= sovBlob(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func NewPopulatedID(r randyBlob, easy bool) *ID {
	this := &ID{}
	this.HashAlg = pb.HashAlg([]int32{0, 1, 2}[r.Intn(3)])
	v1 := redwood_dev_types.NewPopulatedHash(r)
	this.Hash = *v1
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

func NewPopulatedManifest(r randyBlob, easy bool) *Manifest {
	this := &Manifest{}
	this.TotalSize = uint64(uint64(r.Uint32()))
	if r.Intn(5) != 0 {
		v2 := r.Intn(5)
		this.Chunks = make([]ManifestChunk, v2)
		for i := 0; i < v2; i++ {
			v3 := NewPopulatedManifestChunk(r, easy)
			this.Chunks[i] = *v3
		}
	}
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

func NewPopulatedManifestChunk(r randyBlob, easy bool) *ManifestChunk {
	this := &ManifestChunk{}
	v4 := redwood_dev_types.NewPopulatedHash(r)
	this.SHA3 = *v4
	v5 := pb.NewPopulatedRange(r, easy)
	this.Range = *v5
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

type randyBlob interface {
	Float32() float32
	Float64() float64
	Int63() int64
	Int31() int32
	Uint32() uint32
	Intn(n int) int
}

func randUTF8RuneBlob(r randyBlob) rune {
	ru := r.Intn(62)
	if ru < 10 {
		return rune(ru + 48)
	} else if ru < 36 {
		return rune(ru + 55)
	}
	return rune(ru + 61)
}
func randStringBlob(r randyBlob) string {
	v6 := r.Intn(100)
	tmps := make([]rune, v6)
	for i := 0; i < v6; i++ {
		tmps[i] = randUTF8RuneBlob(r)
	}
	return string(tmps)
}
func randUnrecognizedBlob(r randyBlob, maxFieldNumber int) (dAtA []byte) {
	l := r.Intn(5)
	for i := 0; i < l; i++ {
		wire := r.Intn(4)
		if wire == 3 {
			wire = 5
		}
		fieldNumber := maxFieldNumber + r.Intn(100)
		dAtA = randFieldBlob(dAtA, r, fieldNumber, wire)
	}
	return dAtA
}
func randFieldBlob(dAtA []byte, r randyBlob, fieldNumber int, wire int) []byte {
	key := uint32(fieldNumber)<<3 | uint32(wire)
	switch wire {
	case 0:
		dAtA = encodeVarintPopulateBlob(dAtA, uint64(key))
		v7 := r.Int63()
		if r.Intn(2) == 0 {
			v7 *= -1
		}
		dAtA = encodeVarintPopulateBlob(dAtA, uint64(v7))
	case 1:
		dAtA = encodeVarintPopulateBlob(dAtA, uint64(key))
		dAtA = append(dAtA, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	case 2:
		dAtA = encodeVarintPopulateBlob(dAtA, uint64(key))
		ll := r.Intn(100)
		dAtA = encodeVarintPopulateBlob(dAtA, uint64(ll))
		for j := 0; j < ll; j++ {
			dAtA = append(dAtA, byte(r.Intn(256)))
		}
	default:
		dAtA = encodeVarintPopulateBlob(dAtA, uint64(key))
		dAtA = append(dAtA, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	}
	return dAtA
}
func encodeVarintPopulateBlob(dAtA []byte, v uint64) []byte {
	for v >= 1<<7 {
		dAtA = append(dAtA, uint8(uint64(v)&0x7f|0x80))
		v >>= 7
	}
	dAtA = append(dAtA, uint8(v))
	return dAtA
}
func (m *ID) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.HashAlg != 0 {
		n += 1 + sovBlob(uint64(m.HashAlg))
	}
	l = m.Hash.Size()
	n += 1 + l + sovBlob(uint64(l))
	return n
}

func (m *Manifest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.TotalSize != 0 {
		n += 1 + sovBlob(uint64(m.TotalSize))
	}
	if len(m.Chunks) > 0 {
		for _, e := range m.Chunks {
			l = e.Size()
			n += 1 + l + sovBlob(uint64(l))
		}
	}
	return n
}

func (m *ManifestChunk) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.SHA3.Size()
	n += 1 + l + sovBlob(uint64(l))
	l = m.Range.Size()
	n += 1 + l + sovBlob(uint64(l))
	return n
}

func sovBlob(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozBlob(x uint64) (n int) {
	return sovBlob(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Manifest) String() string {
	if this == nil {
		return "nil"
	}
	repeatedStringForChunks := "[]ManifestChunk{"
	for _, f := range this.Chunks {
		repeatedStringForChunks += strings.Replace(strings.Replace(f.String(), "ManifestChunk", "ManifestChunk", 1), `&`, ``, 1) + ","
	}
	repeatedStringForChunks += "}"
	s := strings.Join([]string{`&Manifest{`,
		`TotalSize:` + fmt.Sprintf("%v", this.TotalSize) + `,`,
		`Chunks:` + repeatedStringForChunks + `,`,
		`}`,
	}, "")
	return s
}
func (this *ManifestChunk) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&ManifestChunk{`,
		`SHA3:` + fmt.Sprintf("%v", this.SHA3) + `,`,
		`Range:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.Range), "Range", "pb.Range", 1), `&`, ``, 1) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringBlob(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *ID) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlob
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ID: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ID: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field HashAlg", wireType)
			}
			m.HashAlg = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.HashAlg |= pb.HashAlg(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Hash.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBlob(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlob
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Manifest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlob
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Manifest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Manifest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalSize", wireType)
			}
			m.TotalSize = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TotalSize |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Chunks", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Chunks = append(m.Chunks, ManifestChunk{})
			if err := m.Chunks[len(m.Chunks)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBlob(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlob
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ManifestChunk) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlob
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ManifestChunk: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ManifestChunk: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SHA3", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.SHA3.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Range", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Range.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBlob(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlob
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipBlob(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowBlob
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthBlob
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupBlob
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthBlob
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthBlob        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowBlob          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupBlob = fmt.Errorf("proto: unexpected end of group")
)
