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
// 	protoc-gen-go v1.36.6
// 	protoc        v3.21.12
// source: committee_certificate.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// A certificate proofing the validity of a committee for a specific period.
type CommitteeCertificate struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The aggregated signature of the preceding period's committee, certifying
	// the current committee.
	Signature *AggregatedSignature `protobuf:"bytes,1,opt,name=signature,proto3" json:"signature,omitempty"`
	// The chain ID of the chain the committee is active on.
	ChainId uint64 `protobuf:"varint,2,opt,name=chainId,proto3" json:"chainId,omitempty"`
	// The period in which the committee is active.
	Period uint64 `protobuf:"varint,3,opt,name=period,proto3" json:"period,omitempty"`
	// The members of the committee.
	Members       []*Member `protobuf:"bytes,4,rep,name=members,proto3" json:"members,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CommitteeCertificate) Reset() {
	*x = CommitteeCertificate{}
	mi := &file_committee_certificate_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CommitteeCertificate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommitteeCertificate) ProtoMessage() {}

func (x *CommitteeCertificate) ProtoReflect() protoreflect.Message {
	mi := &file_committee_certificate_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommitteeCertificate.ProtoReflect.Descriptor instead.
func (*CommitteeCertificate) Descriptor() ([]byte, []int) {
	return file_committee_certificate_proto_rawDescGZIP(), []int{0}
}

func (x *CommitteeCertificate) GetSignature() *AggregatedSignature {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *CommitteeCertificate) GetChainId() uint64 {
	if x != nil {
		return x.ChainId
	}
	return 0
}

func (x *CommitteeCertificate) GetPeriod() uint64 {
	if x != nil {
		return x.Period
	}
	return 0
}

func (x *CommitteeCertificate) GetMembers() []*Member {
	if x != nil {
		return x.Members
	}
	return nil
}

// A member of the committee.
type Member struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The 48-byte BLS public key of the member.
	PublicKey []byte `protobuf:"bytes,1,opt,name=publicKey,proto3" json:"publicKey,omitempty"`
	// The 96-byte proof of possession of the private key, proofing that at some
	// point in time the member had access to the private key corresponding to
	// the public key.
	ProofOfPossession []byte `protobuf:"bytes,2,opt,name=proofOfPossession,proto3" json:"proofOfPossession,omitempty"`
	// The voting power of the member. This value defines the weight of the
	// member's vote in the committee.
	VotingPower   uint64 `protobuf:"varint,3,opt,name=votingPower,proto3" json:"votingPower,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Member) Reset() {
	*x = Member{}
	mi := &file_committee_certificate_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Member) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Member) ProtoMessage() {}

func (x *Member) ProtoReflect() protoreflect.Message {
	mi := &file_committee_certificate_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Member.ProtoReflect.Descriptor instead.
func (*Member) Descriptor() ([]byte, []int) {
	return file_committee_certificate_proto_rawDescGZIP(), []int{1}
}

func (x *Member) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *Member) GetProofOfPossession() []byte {
	if x != nil {
		return x.ProofOfPossession
	}
	return nil
}

func (x *Member) GetVotingPower() uint64 {
	if x != nil {
		return x.VotingPower
	}
	return 0
}

var File_committee_certificate_proto protoreflect.FileDescriptor

const file_committee_certificate_proto_rawDesc = "" +
	"\n" +
	"\x1bcommittee_certificate.proto\x12\x14sonic.scc.cert.proto\x1a\x0fsignature.proto\"\xc9\x01\n" +
	"\x14CommitteeCertificate\x12G\n" +
	"\tsignature\x18\x01 \x01(\v2).sonic.scc.cert.proto.AggregatedSignatureR\tsignature\x12\x18\n" +
	"\achainId\x18\x02 \x01(\x04R\achainId\x12\x16\n" +
	"\x06period\x18\x03 \x01(\x04R\x06period\x126\n" +
	"\amembers\x18\x04 \x03(\v2\x1c.sonic.scc.cert.proto.MemberR\amembers\"v\n" +
	"\x06Member\x12\x1c\n" +
	"\tpublicKey\x18\x01 \x01(\fR\tpublicKey\x12,\n" +
	"\x11proofOfPossession\x18\x02 \x01(\fR\x11proofOfPossession\x12 \n" +
	"\vvotingPower\x18\x03 \x01(\x04R\vvotingPowerB\x06Z\x04.;pbb\x06proto3"

var (
	file_committee_certificate_proto_rawDescOnce sync.Once
	file_committee_certificate_proto_rawDescData []byte
)

func file_committee_certificate_proto_rawDescGZIP() []byte {
	file_committee_certificate_proto_rawDescOnce.Do(func() {
		file_committee_certificate_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_committee_certificate_proto_rawDesc), len(file_committee_certificate_proto_rawDesc)))
	})
	return file_committee_certificate_proto_rawDescData
}

var file_committee_certificate_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_committee_certificate_proto_goTypes = []any{
	(*CommitteeCertificate)(nil), // 0: sonic.scc.cert.proto.CommitteeCertificate
	(*Member)(nil),               // 1: sonic.scc.cert.proto.Member
	(*AggregatedSignature)(nil),  // 2: sonic.scc.cert.proto.AggregatedSignature
}
var file_committee_certificate_proto_depIdxs = []int32{
	2, // 0: sonic.scc.cert.proto.CommitteeCertificate.signature:type_name -> sonic.scc.cert.proto.AggregatedSignature
	1, // 1: sonic.scc.cert.proto.CommitteeCertificate.members:type_name -> sonic.scc.cert.proto.Member
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_committee_certificate_proto_init() }
func file_committee_certificate_proto_init() {
	if File_committee_certificate_proto != nil {
		return
	}
	file_signature_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_committee_certificate_proto_rawDesc), len(file_committee_certificate_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_committee_certificate_proto_goTypes,
		DependencyIndexes: file_committee_certificate_proto_depIdxs,
		MessageInfos:      file_committee_certificate_proto_msgTypes,
	}.Build()
	File_committee_certificate_proto = out.File
	file_committee_certificate_proto_goTypes = nil
	file_committee_certificate_proto_depIdxs = nil
}
