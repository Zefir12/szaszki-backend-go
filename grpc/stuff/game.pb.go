// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.31.1
// source: proto/game.proto

package auth

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

type Move struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	From          int32                  `protobuf:"varint,1,opt,name=from,proto3" json:"from,omitempty"`
	To            int32                  `protobuf:"varint,2,opt,name=to,proto3" json:"to,omitempty"`
	Promotion     int32                  `protobuf:"varint,3,opt,name=promotion,proto3" json:"promotion,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Move) Reset() {
	*x = Move{}
	mi := &file_proto_game_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Move) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Move) ProtoMessage() {}

func (x *Move) ProtoReflect() protoreflect.Message {
	mi := &file_proto_game_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Move.ProtoReflect.Descriptor instead.
func (*Move) Descriptor() ([]byte, []int) {
	return file_proto_game_proto_rawDescGZIP(), []int{0}
}

func (x *Move) GetFrom() int32 {
	if x != nil {
		return x.From
	}
	return 0
}

func (x *Move) GetTo() int32 {
	if x != nil {
		return x.To
	}
	return 0
}

func (x *Move) GetPromotion() int32 {
	if x != nil {
		return x.Promotion
	}
	return 0
}

type GameState struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BoardHistory  [][]byte               `protobuf:"bytes,1,rep,name=board_history,json=boardHistory,proto3" json:"board_history,omitempty"`
	MoveHistory   []*Move                `protobuf:"bytes,2,rep,name=move_history,json=moveHistory,proto3" json:"move_history,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GameState) Reset() {
	*x = GameState{}
	mi := &file_proto_game_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GameState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GameState) ProtoMessage() {}

func (x *GameState) ProtoReflect() protoreflect.Message {
	mi := &file_proto_game_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GameState.ProtoReflect.Descriptor instead.
func (*GameState) Descriptor() ([]byte, []int) {
	return file_proto_game_proto_rawDescGZIP(), []int{1}
}

func (x *GameState) GetBoardHistory() [][]byte {
	if x != nil {
		return x.BoardHistory
	}
	return nil
}

func (x *GameState) GetMoveHistory() []*Move {
	if x != nil {
		return x.MoveHistory
	}
	return nil
}

type SaveGameRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	GameId        uint32                 `protobuf:"varint,1,opt,name=game_id,json=gameId,proto3" json:"game_id,omitempty"`
	UserIdWhite   uint32                 `protobuf:"varint,2,opt,name=user_id_white,json=userIdWhite,proto3" json:"user_id_white,omitempty"`
	UserIdBlack   uint32                 `protobuf:"varint,3,opt,name=user_id_black,json=userIdBlack,proto3" json:"user_id_black,omitempty"`
	GameState     *GameState             `protobuf:"bytes,4,opt,name=game_state,json=gameState,proto3" json:"game_state,omitempty"`
	Pgn           string                 `protobuf:"bytes,5,opt,name=pgn,proto3" json:"pgn,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SaveGameRequest) Reset() {
	*x = SaveGameRequest{}
	mi := &file_proto_game_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SaveGameRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaveGameRequest) ProtoMessage() {}

func (x *SaveGameRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_game_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SaveGameRequest.ProtoReflect.Descriptor instead.
func (*SaveGameRequest) Descriptor() ([]byte, []int) {
	return file_proto_game_proto_rawDescGZIP(), []int{2}
}

func (x *SaveGameRequest) GetGameId() uint32 {
	if x != nil {
		return x.GameId
	}
	return 0
}

func (x *SaveGameRequest) GetUserIdWhite() uint32 {
	if x != nil {
		return x.UserIdWhite
	}
	return 0
}

func (x *SaveGameRequest) GetUserIdBlack() uint32 {
	if x != nil {
		return x.UserIdBlack
	}
	return 0
}

func (x *SaveGameRequest) GetGameState() *GameState {
	if x != nil {
		return x.GameState
	}
	return nil
}

func (x *SaveGameRequest) GetPgn() string {
	if x != nil {
		return x.Pgn
	}
	return ""
}

type SaveGameResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message       string                 `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SaveGameResponse) Reset() {
	*x = SaveGameResponse{}
	mi := &file_proto_game_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SaveGameResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaveGameResponse) ProtoMessage() {}

func (x *SaveGameResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_game_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SaveGameResponse.ProtoReflect.Descriptor instead.
func (*SaveGameResponse) Descriptor() ([]byte, []int) {
	return file_proto_game_proto_rawDescGZIP(), []int{3}
}

func (x *SaveGameResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *SaveGameResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_proto_game_proto protoreflect.FileDescriptor

const file_proto_game_proto_rawDesc = "" +
	"\n" +
	"\x10proto/game.proto\x12\x04game\"H\n" +
	"\x04Move\x12\x12\n" +
	"\x04from\x18\x01 \x01(\x05R\x04from\x12\x0e\n" +
	"\x02to\x18\x02 \x01(\x05R\x02to\x12\x1c\n" +
	"\tpromotion\x18\x03 \x01(\x05R\tpromotion\"_\n" +
	"\tGameState\x12#\n" +
	"\rboard_history\x18\x01 \x03(\fR\fboardHistory\x12-\n" +
	"\fmove_history\x18\x02 \x03(\v2\n" +
	".game.MoveR\vmoveHistory\"\xb4\x01\n" +
	"\x0fSaveGameRequest\x12\x17\n" +
	"\agame_id\x18\x01 \x01(\rR\x06gameId\x12\"\n" +
	"\ruser_id_white\x18\x02 \x01(\rR\vuserIdWhite\x12\"\n" +
	"\ruser_id_black\x18\x03 \x01(\rR\vuserIdBlack\x12.\n" +
	"\n" +
	"game_state\x18\x04 \x01(\v2\x0f.game.GameStateR\tgameState\x12\x10\n" +
	"\x03pgn\x18\x05 \x01(\tR\x03pgn\"F\n" +
	"\x10SaveGameResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x18\n" +
	"\amessage\x18\x02 \x01(\tR\amessage2J\n" +
	"\vGameService\x12;\n" +
	"\bSaveGame\x12\x15.game.SaveGameRequest\x1a\x16.game.SaveGameResponse\"\x00B\x12Z\x10/grpc/stuff;authb\x06proto3"

var (
	file_proto_game_proto_rawDescOnce sync.Once
	file_proto_game_proto_rawDescData []byte
)

func file_proto_game_proto_rawDescGZIP() []byte {
	file_proto_game_proto_rawDescOnce.Do(func() {
		file_proto_game_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_game_proto_rawDesc), len(file_proto_game_proto_rawDesc)))
	})
	return file_proto_game_proto_rawDescData
}

var file_proto_game_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_proto_game_proto_goTypes = []any{
	(*Move)(nil),             // 0: game.Move
	(*GameState)(nil),        // 1: game.GameState
	(*SaveGameRequest)(nil),  // 2: game.SaveGameRequest
	(*SaveGameResponse)(nil), // 3: game.SaveGameResponse
}
var file_proto_game_proto_depIdxs = []int32{
	0, // 0: game.GameState.move_history:type_name -> game.Move
	1, // 1: game.SaveGameRequest.game_state:type_name -> game.GameState
	2, // 2: game.GameService.SaveGame:input_type -> game.SaveGameRequest
	3, // 3: game.GameService.SaveGame:output_type -> game.SaveGameResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proto_game_proto_init() }
func file_proto_game_proto_init() {
	if File_proto_game_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_game_proto_rawDesc), len(file_proto_game_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_game_proto_goTypes,
		DependencyIndexes: file_proto_game_proto_depIdxs,
		MessageInfos:      file_proto_game_proto_msgTypes,
	}.Build()
	File_proto_game_proto = out.File
	file_proto_game_proto_goTypes = nil
	file_proto_game_proto_depIdxs = nil
}
