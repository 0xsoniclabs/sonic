// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.14.0
// source: signature.proto

package pb

import (
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

// The aggregated signature of multiple signers.
type AggregatedSignature struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The 96-byte BLS signature aggregating the signatures of all signers.
	Signature []byte `protobuf:"bytes,1,opt,name=signature,proto3" json:"signature,omitempty"`
	// A bit-mask indicating which signers have signed the block.
	// The i-th bit of the mask corresponds to the i-th signer.
	// If the i-th bit is set, the i-th signer has signed the block.
	SignerMask []byte `protobuf:"bytes,2,opt,name=signerMask,proto3" json:"signerMask,omitempty"`
}

func (x *AggregatedSignature) Reset() {
	*x = AggregatedSignature{}
	if protoimpl.UnsafeEnabled {
		mi := &file_signature_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AggregatedSignature) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AggregatedSignature) ProtoMessage() {}

func (x *AggregatedSignature) ProtoReflect() protoreflect.Message {
	mi := &file_signature_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AggregatedSignature.ProtoReflect.Descriptor instead.
func (*AggregatedSignature) Descriptor() ([]byte, []int) {
	return file_signature_proto_rawDescGZIP(), []int{0}
}

func (x *AggregatedSignature) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *AggregatedSignature) GetSignerMask() []byte {
	if x != nil {
		return x.SignerMask
	}
	return nil
}

var File_signature_proto protoreflect.FileDescriptor

var file_signature_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x14, 0x73, 0x6f, 0x6e, 0x69, 0x63, 0x2e, 0x73, 0x63, 0x63, 0x2e, 0x63, 0x65, 0x72,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x53, 0x0a, 0x13, 0x41, 0x67, 0x67, 0x72, 0x65,
	0x67, 0x61, 0x74, 0x65, 0x64, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1c,
	0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1e, 0x0a, 0x0a,
	0x73, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x4d, 0x61, 0x73, 0x6b, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x0a, 0x73, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x4d, 0x61, 0x73, 0x6b, 0x42, 0x06, 0x5a, 0x04,
	0x2e, 0x3b, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_signature_proto_rawDescOnce sync.Once
	file_signature_proto_rawDescData = file_signature_proto_rawDesc
)

func file_signature_proto_rawDescGZIP() []byte {
	file_signature_proto_rawDescOnce.Do(func() {
		file_signature_proto_rawDescData = protoimpl.X.CompressGZIP(file_signature_proto_rawDescData)
	})
	return file_signature_proto_rawDescData
}

var file_signature_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_signature_proto_goTypes = []interface{}{
	(*AggregatedSignature)(nil), // 0: sonic.scc.cert.proto.AggregatedSignature
}
var file_signature_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_signature_proto_init() }
func file_signature_proto_init() {
	if File_signature_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_signature_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AggregatedSignature); i {
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
			RawDescriptor: file_signature_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_signature_proto_goTypes,
		DependencyIndexes: file_signature_proto_depIdxs,
		MessageInfos:      file_signature_proto_msgTypes,
	}.Build()
	File_signature_proto = out.File
	file_signature_proto_rawDesc = nil
	file_signature_proto_goTypes = nil
	file_signature_proto_depIdxs = nil
}
