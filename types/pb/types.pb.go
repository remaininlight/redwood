// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: types.proto

package pb

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
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

type HashAlg int32

const (
	HashAlgUnknown HashAlg = 0
	SHA1           HashAlg = 1
	SHA3           HashAlg = 2
)

var HashAlg_name = map[int32]string{
	0: "Unknown",
	1: "SHA1",
	2: "SHA3",
}

var HashAlg_value = map[string]int32{
	"Unknown": 0,
	"SHA1":    1,
	"SHA3":    2,
}

func (HashAlg) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{0}
}

type Range struct {
	Start   uint64 `protobuf:"varint,1,opt,name=start,proto3" json:"start" tree:"start"`
	End     uint64 `protobuf:"varint,2,opt,name=end,proto3" json:"end" tree:"end"`
	Reverse bool   `protobuf:"varint,3,opt,name=reverse,proto3" json:"reverse" tree:"reverse"`
}

func (m *Range) Reset()      { *m = Range{} }
func (*Range) ProtoMessage() {}
func (*Range) Descriptor() ([]byte, []int) {
	return fileDescriptor_d938547f84707355, []int{0}
}
func (m *Range) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Range) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Range.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Range) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Range.Merge(m, src)
}
func (m *Range) XXX_Size() int {
	return m.Size()
}
func (m *Range) XXX_DiscardUnknown() {
	xxx_messageInfo_Range.DiscardUnknown(m)
}

var xxx_messageInfo_Range proto.InternalMessageInfo

func (m *Range) GetStart() uint64 {
	if m != nil {
		return m.Start
	}
	return 0
}

func (m *Range) GetEnd() uint64 {
	if m != nil {
		return m.End
	}
	return 0
}

func (m *Range) GetReverse() bool {
	if m != nil {
		return m.Reverse
	}
	return false
}

func init() {
	proto.RegisterEnum("Redwood.types.HashAlg", HashAlg_name, HashAlg_value)
	proto.RegisterType((*Range)(nil), "Redwood.types.Range")
}

func init() { proto.RegisterFile("types.proto", fileDescriptor_d938547f84707355) }

var fileDescriptor_d938547f84707355 = []byte{
	// 334 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x34, 0x90, 0x3f, 0x4c, 0xc2, 0x40,
	0x14, 0xc6, 0xef, 0xf1, 0x47, 0xc8, 0xa9, 0x84, 0x5c, 0x34, 0x41, 0x12, 0xdf, 0x91, 0x2e, 0x12,
	0x13, 0x69, 0x0c, 0x83, 0x89, 0x1b, 0xc4, 0x81, 0xb9, 0xc6, 0x85, 0x8d, 0xda, 0xb3, 0x18, 0xb5,
	0x47, 0xda, 0x02, 0x71, 0x73, 0x76, 0x72, 0x36, 0x71, 0x73, 0x70, 0x74, 0x74, 0x74, 0x74, 0x64,
	0x64, 0x6a, 0xe8, 0xb1, 0x18, 0x26, 0xc2, 0xe4, 0x68, 0xb8, 0x96, 0xed, 0xfd, 0x7e, 0xef, 0xfb,
	0x96, 0x8f, 0x6e, 0x87, 0x8f, 0x03, 0x11, 0x34, 0x06, 0xbe, 0x0c, 0x25, 0xdb, 0xb5, 0x84, 0x33,
	0x96, 0xd2, 0x69, 0x68, 0x59, 0x3d, 0x71, 0x6f, 0xc3, 0xfe, 0xd0, 0x6e, 0x5c, 0xcb, 0x07, 0xd3,
	0x95, 0xae, 0x34, 0x75, 0xca, 0x1e, 0xde, 0x68, 0xd2, 0xa0, 0xaf, 0xa4, 0x6d, 0xbc, 0x02, 0xcd,
	0x5b, 0x3d, 0xcf, 0x15, 0xcc, 0xa4, 0xf9, 0x20, 0xec, 0xf9, 0x61, 0x05, 0x6a, 0x50, 0xcf, 0xb5,
	0x0f, 0x16, 0x11, 0x4f, 0xc4, 0x2a, 0xe2, 0x3b, 0xa1, 0x2f, 0xc4, 0xb9, 0xa1, 0xd1, 0xb0, 0x12,
	0xcd, 0x8e, 0x68, 0x56, 0x78, 0x4e, 0x25, 0xa3, 0xe3, 0xfb, 0x8b, 0x88, 0xaf, 0x71, 0x15, 0x71,
	0x9a, 0x84, 0x85, 0xe7, 0x18, 0xd6, 0x5a, 0xb1, 0x33, 0x5a, 0xf0, 0xc5, 0x48, 0xf8, 0x81, 0xa8,
	0x64, 0x6b, 0x50, 0x2f, 0xb6, 0x0f, 0x17, 0x11, 0xdf, 0xa8, 0x55, 0xc4, 0x4b, 0x49, 0x21, 0x15,
	0x86, 0xb5, 0x79, 0x1d, 0x5f, 0xd0, 0x42, 0xa7, 0x17, 0xf4, 0x5b, 0xf7, 0x2e, 0xe3, 0xb4, 0x70,
	0xe5, 0xdd, 0x79, 0x72, 0xec, 0x95, 0x49, 0x95, 0x3d, 0xbf, 0xd5, 0x4a, 0xe9, 0x27, 0xb5, 0xac,
	0x48, 0x73, 0x97, 0x9d, 0xd6, 0x69, 0x19, 0xd2, 0xab, 0x59, 0xce, 0x54, 0x73, 0x9f, 0xef, 0x48,
	0xda, 0xdd, 0x49, 0x8c, 0x64, 0x1a, 0x23, 0x99, 0xc5, 0x08, 0xcb, 0x18, 0xe1, 0x2f, 0x46, 0x78,
	0x52, 0x08, 0x1f, 0x0a, 0xe1, 0x4b, 0x21, 0xf9, 0x56, 0x08, 0x3f, 0x0a, 0x61, 0xa2, 0x10, 0x66,
	0x0a, 0xe1, 0x57, 0x21, 0x59, 0x2a, 0x84, 0x97, 0x39, 0x92, 0xc9, 0x1c, 0xc9, 0x74, 0x8e, 0xa4,
	0xbb, 0xe7, 0xa7, 0x33, 0x3b, 0x62, 0x64, 0xea, 0xa9, 0xcd, 0x81, 0x6d, 0x6f, 0xe9, 0x15, 0x9b,
	0xff, 0x01, 0x00, 0x00, 0xff, 0xff, 0x2c, 0x4c, 0x6f, 0xfa, 0x92, 0x01, 0x00, 0x00,
}

func (this *Range) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*Range)
	if !ok {
		that2, ok := that.(Range)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *Range")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *Range but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *Range but is not nil && this == nil")
	}
	if this.Start != that1.Start {
		return fmt.Errorf("Start this(%v) Not Equal that(%v)", this.Start, that1.Start)
	}
	if this.End != that1.End {
		return fmt.Errorf("End this(%v) Not Equal that(%v)", this.End, that1.End)
	}
	if this.Reverse != that1.Reverse {
		return fmt.Errorf("Reverse this(%v) Not Equal that(%v)", this.Reverse, that1.Reverse)
	}
	return nil
}
func (this *Range) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Range)
	if !ok {
		that2, ok := that.(Range)
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
	if this.Start != that1.Start {
		return false
	}
	if this.End != that1.End {
		return false
	}
	if this.Reverse != that1.Reverse {
		return false
	}
	return true
}
func (this *Range) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 7)
	s = append(s, "&pb.Range{")
	s = append(s, "Start: "+fmt.Sprintf("%#v", this.Start)+",\n")
	s = append(s, "End: "+fmt.Sprintf("%#v", this.End)+",\n")
	s = append(s, "Reverse: "+fmt.Sprintf("%#v", this.Reverse)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringTypes(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *Range) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Range) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Range) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Reverse {
		i--
		if m.Reverse {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x18
	}
	if m.End != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.End))
		i--
		dAtA[i] = 0x10
	}
	if m.Start != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Start))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintTypes(dAtA []byte, offset int, v uint64) int {
	offset -= sovTypes(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func NewPopulatedRange(r randyTypes, easy bool) *Range {
	this := &Range{}
	this.Start = uint64(uint64(r.Uint32()))
	this.End = uint64(uint64(r.Uint32()))
	this.Reverse = bool(bool(r.Intn(2) == 0))
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

type randyTypes interface {
	Float32() float32
	Float64() float64
	Int63() int64
	Int31() int32
	Uint32() uint32
	Intn(n int) int
}

func randUTF8RuneTypes(r randyTypes) rune {
	ru := r.Intn(62)
	if ru < 10 {
		return rune(ru + 48)
	} else if ru < 36 {
		return rune(ru + 55)
	}
	return rune(ru + 61)
}
func randStringTypes(r randyTypes) string {
	v1 := r.Intn(100)
	tmps := make([]rune, v1)
	for i := 0; i < v1; i++ {
		tmps[i] = randUTF8RuneTypes(r)
	}
	return string(tmps)
}
func randUnrecognizedTypes(r randyTypes, maxFieldNumber int) (dAtA []byte) {
	l := r.Intn(5)
	for i := 0; i < l; i++ {
		wire := r.Intn(4)
		if wire == 3 {
			wire = 5
		}
		fieldNumber := maxFieldNumber + r.Intn(100)
		dAtA = randFieldTypes(dAtA, r, fieldNumber, wire)
	}
	return dAtA
}
func randFieldTypes(dAtA []byte, r randyTypes, fieldNumber int, wire int) []byte {
	key := uint32(fieldNumber)<<3 | uint32(wire)
	switch wire {
	case 0:
		dAtA = encodeVarintPopulateTypes(dAtA, uint64(key))
		v2 := r.Int63()
		if r.Intn(2) == 0 {
			v2 *= -1
		}
		dAtA = encodeVarintPopulateTypes(dAtA, uint64(v2))
	case 1:
		dAtA = encodeVarintPopulateTypes(dAtA, uint64(key))
		dAtA = append(dAtA, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	case 2:
		dAtA = encodeVarintPopulateTypes(dAtA, uint64(key))
		ll := r.Intn(100)
		dAtA = encodeVarintPopulateTypes(dAtA, uint64(ll))
		for j := 0; j < ll; j++ {
			dAtA = append(dAtA, byte(r.Intn(256)))
		}
	default:
		dAtA = encodeVarintPopulateTypes(dAtA, uint64(key))
		dAtA = append(dAtA, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	}
	return dAtA
}
func encodeVarintPopulateTypes(dAtA []byte, v uint64) []byte {
	for v >= 1<<7 {
		dAtA = append(dAtA, uint8(uint64(v)&0x7f|0x80))
		v >>= 7
	}
	dAtA = append(dAtA, uint8(v))
	return dAtA
}
func (m *Range) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Start != 0 {
		n += 1 + sovTypes(uint64(m.Start))
	}
	if m.End != 0 {
		n += 1 + sovTypes(uint64(m.End))
	}
	if m.Reverse {
		n += 2
	}
	return n
}

func sovTypes(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTypes(x uint64) (n int) {
	return sovTypes(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Range) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Range{`,
		`Start:` + fmt.Sprintf("%v", this.Start) + `,`,
		`End:` + fmt.Sprintf("%v", this.End) + `,`,
		`Reverse:` + fmt.Sprintf("%v", this.Reverse) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringTypes(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Range) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
			return fmt.Errorf("proto: Range: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Range: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Start", wireType)
			}
			m.Start = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Start |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field End", wireType)
			}
			m.End = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.End |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reverse", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Reverse = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTypes
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
func skipTypes(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTypes
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
					return 0, ErrIntOverflowTypes
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
					return 0, ErrIntOverflowTypes
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
				return 0, ErrInvalidLengthTypes
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTypes
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTypes
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTypes        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTypes          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTypes = fmt.Errorf("proto: unexpected end of group")
)
