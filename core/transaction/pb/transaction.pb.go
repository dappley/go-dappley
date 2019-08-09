// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/dappley/go-dappley/core/transaction/pb/transaction.proto

package transactionpb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import pb "github.com/dappley/go-dappley/core/transaction_base/pb"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Transactions struct {
	Transactions         []*Transaction `protobuf:"bytes,1,rep,name=transactions,proto3" json:"transactions,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Transactions) Reset()         { *m = Transactions{} }
func (m *Transactions) String() string { return proto.CompactTextString(m) }
func (*Transactions) ProtoMessage()    {}
func (*Transactions) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_1b496d2674fd2d27, []int{0}
}
func (m *Transactions) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Transactions.Unmarshal(m, b)
}
func (m *Transactions) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Transactions.Marshal(b, m, deterministic)
}
func (dst *Transactions) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Transactions.Merge(dst, src)
}
func (m *Transactions) XXX_Size() int {
	return xxx_messageInfo_Transactions.Size(m)
}
func (m *Transactions) XXX_DiscardUnknown() {
	xxx_messageInfo_Transactions.DiscardUnknown(m)
}

var xxx_messageInfo_Transactions proto.InternalMessageInfo

func (m *Transactions) GetTransactions() []*Transaction {
	if m != nil {
		return m.Transactions
	}
	return nil
}

type Transaction struct {
	Id                   []byte         `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Vin                  []*pb.TXInput  `protobuf:"bytes,2,rep,name=vin,proto3" json:"vin,omitempty"`
	Vout                 []*pb.TXOutput `protobuf:"bytes,3,rep,name=vout,proto3" json:"vout,omitempty"`
	Tip                  []byte         `protobuf:"bytes,4,opt,name=tip,proto3" json:"tip,omitempty"`
	GasLimit             []byte         `protobuf:"bytes,5,opt,name=gas_limit,json=gasLimit,proto3" json:"gas_limit,omitempty"`
	GasPrice             []byte         `protobuf:"bytes,6,opt,name=gas_price,json=gasPrice,proto3" json:"gas_price,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Transaction) Reset()         { *m = Transaction{} }
func (m *Transaction) String() string { return proto.CompactTextString(m) }
func (*Transaction) ProtoMessage()    {}
func (*Transaction) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_1b496d2674fd2d27, []int{1}
}
func (m *Transaction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Transaction.Unmarshal(m, b)
}
func (m *Transaction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Transaction.Marshal(b, m, deterministic)
}
func (dst *Transaction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Transaction.Merge(dst, src)
}
func (m *Transaction) XXX_Size() int {
	return xxx_messageInfo_Transaction.Size(m)
}
func (m *Transaction) XXX_DiscardUnknown() {
	xxx_messageInfo_Transaction.DiscardUnknown(m)
}

var xxx_messageInfo_Transaction proto.InternalMessageInfo

func (m *Transaction) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Transaction) GetVin() []*pb.TXInput {
	if m != nil {
		return m.Vin
	}
	return nil
}

func (m *Transaction) GetVout() []*pb.TXOutput {
	if m != nil {
		return m.Vout
	}
	return nil
}

func (m *Transaction) GetTip() []byte {
	if m != nil {
		return m.Tip
	}
	return nil
}

func (m *Transaction) GetGasLimit() []byte {
	if m != nil {
		return m.GasLimit
	}
	return nil
}

func (m *Transaction) GetGasPrice() []byte {
	if m != nil {
		return m.GasPrice
	}
	return nil
}

type TransactionNode struct {
	Children             map[string]*Transaction `protobuf:"bytes,1,rep,name=children,proto3" json:"children,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Value                *Transaction            `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Size                 int64                   `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *TransactionNode) Reset()         { *m = TransactionNode{} }
func (m *TransactionNode) String() string { return proto.CompactTextString(m) }
func (*TransactionNode) ProtoMessage()    {}
func (*TransactionNode) Descriptor() ([]byte, []int) {
	return fileDescriptor_transaction_1b496d2674fd2d27, []int{2}
}
func (m *TransactionNode) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TransactionNode.Unmarshal(m, b)
}
func (m *TransactionNode) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TransactionNode.Marshal(b, m, deterministic)
}
func (dst *TransactionNode) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransactionNode.Merge(dst, src)
}
func (m *TransactionNode) XXX_Size() int {
	return xxx_messageInfo_TransactionNode.Size(m)
}
func (m *TransactionNode) XXX_DiscardUnknown() {
	xxx_messageInfo_TransactionNode.DiscardUnknown(m)
}

var xxx_messageInfo_TransactionNode proto.InternalMessageInfo

func (m *TransactionNode) GetChildren() map[string]*Transaction {
	if m != nil {
		return m.Children
	}
	return nil
}

func (m *TransactionNode) GetValue() *Transaction {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *TransactionNode) GetSize() int64 {
	if m != nil {
		return m.Size
	}
	return 0
}

func init() {
	proto.RegisterType((*Transactions)(nil), "transactionpb.Transactions")
	proto.RegisterType((*Transaction)(nil), "transactionpb.Transaction")
	proto.RegisterType((*TransactionNode)(nil), "transactionpb.TransactionNode")
	proto.RegisterMapType((map[string]*Transaction)(nil), "transactionpb.TransactionNode.ChildrenEntry")
}

func init() {
	proto.RegisterFile("github.com/dappley/go-dappley/core/transaction/pb/transaction.proto", fileDescriptor_transaction_1b496d2674fd2d27)
}

var fileDescriptor_transaction_1b496d2674fd2d27 = []byte{
	// 346 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0x4f, 0x4b, 0xc3, 0x40,
	0x10, 0xc5, 0xd9, 0xa4, 0x2d, 0xed, 0xb4, 0x55, 0xd9, 0xd3, 0xd2, 0x5e, 0x42, 0x4f, 0x3d, 0xd4,
	0x44, 0xf4, 0x22, 0x1e, 0x3c, 0x58, 0x04, 0x85, 0x52, 0x25, 0x08, 0x7a, 0x2b, 0x9b, 0x64, 0x49,
	0x17, 0xd3, 0xec, 0x92, 0xdd, 0x14, 0xea, 0x27, 0xf4, 0x33, 0x79, 0x92, 0xdd, 0xfe, 0x31, 0xf5,
	0x0f, 0xd2, 0xdb, 0xcc, 0xbe, 0xdf, 0xbc, 0x99, 0x3c, 0x02, 0xe3, 0x94, 0xeb, 0x79, 0x19, 0xf9,
	0xb1, 0x58, 0x04, 0x09, 0x95, 0x32, 0x63, 0xab, 0x20, 0x15, 0xa7, 0xdb, 0x32, 0x16, 0x05, 0x0b,
	0x74, 0x41, 0x73, 0x45, 0x63, 0xcd, 0x45, 0x1e, 0xc8, 0xa8, 0xda, 0xfa, 0xb2, 0x10, 0x5a, 0xe0,
	0x6e, 0xe5, 0x49, 0x46, 0xbd, 0xc9, 0x61, 0x9e, 0xb3, 0x88, 0x2a, 0xf6, 0xcd, 0xf8, 0x86, 0x2a,
	0xb6, 0x36, 0x1f, 0x4c, 0xa1, 0xf3, 0xf4, 0x25, 0x28, 0x7c, 0x0d, 0x9d, 0x0a, 0xa8, 0x08, 0xf2,
	0xdc, 0x61, 0xfb, 0xbc, 0xe7, 0xef, 0xdd, 0xe0, 0x57, 0x46, 0xc2, 0x3d, 0x7e, 0xf0, 0x8e, 0xa0,
	0x5d, 0x51, 0xf1, 0x11, 0x38, 0x3c, 0x21, 0xc8, 0x43, 0xc3, 0x4e, 0xe8, 0xf0, 0x04, 0x8f, 0xc0,
	0x5d, 0xf2, 0x9c, 0x38, 0x3f, 0x6d, 0xcd, 0x9d, 0xc6, 0xfa, 0xe5, 0x3e, 0x97, 0xa5, 0x0e, 0x0d,
	0x86, 0x03, 0xa8, 0x2d, 0x45, 0xa9, 0x89, 0x6b, 0xf1, 0xfe, 0xaf, 0xf8, 0x43, 0xa9, 0x0d, 0x6f,
	0x41, 0x7c, 0x02, 0xae, 0xe6, 0x92, 0xd4, 0xec, 0x3e, 0x53, 0xe2, 0x3e, 0xb4, 0x52, 0xaa, 0x66,
	0x19, 0x5f, 0x70, 0x4d, 0xea, 0xf6, 0xbd, 0x99, 0x52, 0x35, 0x31, 0xfd, 0x56, 0x94, 0x05, 0x8f,
	0x19, 0x69, 0xec, 0xc4, 0x47, 0xd3, 0x0f, 0x3e, 0x10, 0x1c, 0x57, 0x3e, 0x65, 0x2a, 0x12, 0x86,
	0xef, 0xa0, 0x19, 0xcf, 0x79, 0x96, 0x14, 0x2c, 0xdf, 0x44, 0x33, 0xfa, 0x3b, 0x1a, 0x33, 0xe1,
	0x8f, 0x37, 0xf8, 0x6d, 0xae, 0x8b, 0x55, 0xb8, 0x9b, 0xc6, 0x67, 0x50, 0x5f, 0xd2, 0xac, 0x64,
	0xc4, 0xf1, 0xd0, 0x3f, 0x09, 0xaf, 0x41, 0x8c, 0xa1, 0xa6, 0xf8, 0x1b, 0x23, 0xae, 0x87, 0x86,
	0x6e, 0x68, 0xeb, 0xde, 0x33, 0x74, 0xf7, 0x16, 0x98, 0x00, 0x5e, 0xd9, 0xca, 0x06, 0xde, 0x0a,
	0x4d, 0x79, 0xf8, 0xa2, 0x2b, 0xe7, 0x12, 0x45, 0x0d, 0xfb, 0x7b, 0x5c, 0x7c, 0x06, 0x00, 0x00,
	0xff, 0xff, 0x00, 0x6f, 0x14, 0xef, 0xc2, 0x02, 0x00, 0x00,
}