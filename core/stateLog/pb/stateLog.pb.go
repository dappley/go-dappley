// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.4
// source: stateLog.proto

package stateLogpb

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type Log struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Log map[string]string `protobuf:"bytes,1,rep,name=log,proto3" json:"log,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Log) Reset() {
	*x = Log{}
	if protoimpl.UnsafeEnabled {
		mi := &file_stateLog_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Log) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Log) ProtoMessage() {}

func (x *Log) ProtoReflect() protoreflect.Message {
	mi := &file_stateLog_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Log.ProtoReflect.Descriptor instead.
func (*Log) Descriptor() ([]byte, []int) {
	return file_stateLog_proto_rawDescGZIP(), []int{0}
}

func (x *Log) GetLog() map[string]string {
	if x != nil {
		return x.Log
	}
	return nil
}

type StateLog struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Log map[string]*Log `protobuf:"bytes,1,rep,name=log,proto3" json:"log,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *StateLog) Reset() {
	*x = StateLog{}
	if protoimpl.UnsafeEnabled {
		mi := &file_stateLog_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateLog) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateLog) ProtoMessage() {}

func (x *StateLog) ProtoReflect() protoreflect.Message {
	mi := &file_stateLog_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateLog.ProtoReflect.Descriptor instead.
func (*StateLog) Descriptor() ([]byte, []int) {
	return file_stateLog_proto_rawDescGZIP(), []int{1}
}

func (x *StateLog) GetLog() map[string]*Log {
	if x != nil {
		return x.Log
	}
	return nil
}

var File_stateLog_proto protoreflect.FileDescriptor

var file_stateLog_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x70, 0x62, 0x22, 0x69, 0x0a, 0x03,
	0x4c, 0x6f, 0x67, 0x12, 0x2a, 0x0a, 0x03, 0x6c, 0x6f, 0x67, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x18, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x70, 0x62, 0x2e, 0x4c, 0x6f,
	0x67, 0x2e, 0x4c, 0x6f, 0x67, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x03, 0x6c, 0x6f, 0x67, 0x1a,
	0x36, 0x0a, 0x08, 0x4c, 0x6f, 0x67, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x84, 0x01, 0x0a, 0x08, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x4c, 0x6f, 0x67, 0x12, 0x2f, 0x0a, 0x03, 0x6c, 0x6f, 0x67, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x1d, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x70, 0x62, 0x2e, 0x53,
	0x74, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x2e, 0x4c, 0x6f, 0x67, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x03, 0x6c, 0x6f, 0x67, 0x1a, 0x47, 0x0a, 0x08, 0x4c, 0x6f, 0x67, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x25, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x4c, 0x6f, 0x67, 0x70, 0x62, 0x2e,
	0x4c, 0x6f, 0x67, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_stateLog_proto_rawDescOnce sync.Once
	file_stateLog_proto_rawDescData = file_stateLog_proto_rawDesc
)

func file_stateLog_proto_rawDescGZIP() []byte {
	file_stateLog_proto_rawDescOnce.Do(func() {
		file_stateLog_proto_rawDescData = protoimpl.X.CompressGZIP(file_stateLog_proto_rawDescData)
	})
	return file_stateLog_proto_rawDescData
}

var file_stateLog_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_stateLog_proto_goTypes = []interface{}{
	(*Log)(nil),      // 0: stateLogpb.Log
	(*StateLog)(nil), // 1: stateLogpb.StateLog
	nil,              // 2: stateLogpb.Log.LogEntry
	nil,              // 3: stateLogpb.StateLog.LogEntry
}
var file_stateLog_proto_depIdxs = []int32{
	2, // 0: stateLogpb.Log.log:type_name -> stateLogpb.Log.LogEntry
	3, // 1: stateLogpb.StateLog.log:type_name -> stateLogpb.StateLog.LogEntry
	0, // 2: stateLogpb.StateLog.LogEntry.value:type_name -> stateLogpb.Log
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_stateLog_proto_init() }
func file_stateLog_proto_init() {
	if File_stateLog_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_stateLog_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Log); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_stateLog_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateLog); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_stateLog_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_stateLog_proto_goTypes,
		DependencyIndexes: file_stateLog_proto_depIdxs,
		MessageInfos:      file_stateLog_proto_msgTypes,
	}.Build()
	File_stateLog_proto = out.File
	file_stateLog_proto_rawDesc = nil
	file_stateLog_proto_goTypes = nil
	file_stateLog_proto_depIdxs = nil
}
