package agorartm

/*
#include "C_IAgoraRtmClient.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// IsValidMemory is a nil check only. Native callback memory is borrowed for the
// duration of the callback. Page probes cannot establish ownership or safety and
// process-wide SIGSEGV handlers are incompatible with concurrent Go execution.
func IsValidMemory(ptr unsafe.Pointer) bool { return ptr != nil }
func FastSafeCGoString(cstr *C.char) string { return C.GoString(cstr) }
func SafeCGoString(cstr *C.char) string     { return C.GoString(cstr) }

// nativeSlice checks nil and length overflow before borrowing a native array.
// Callers must deep-copy pointer-containing elements before the callback ends.
func nativeSlice[T any](ptr *T, count C.size_t) []T {
	if ptr == nil || count == 0 {
		return nil
	}
	var element T
	size := unsafe.Sizeof(element)
	if size == 0 || uint64(count) > uint64(int(^uint(0)>>1))/uint64(size) {
		return nil
	}
	return unsafe.Slice(ptr, int(count))
}

func copyStrings(ptr **C.char, count C.size_t) []string {
	src := nativeSlice(ptr, count)
	result := make([]string, len(src))
	for i := range src {
		result[i] = C.GoString(src[i])
	}
	return result
}

func CUserListToUserList(src *C.struct_C_UserList) *UserList {
	if src == nil {
		return nil
	}
	return &UserList{Users: copyStrings(src.users, src.userCount)}
}

func CMetadataToIMetadata(src *C.struct_C_Metadata) *IMetadata {
	if src == nil {
		return nil
	}
	items := nativeSlice(src.items, src.itemCount)
	dst := &IMetadata{majorRevision: int64(src.majorRevision), items: make([]MetadataItem, len(items)), itemCount: uint(len(items))}
	for i := range items {
		item := &items[i]
		dst.items[i] = MetadataItem{Key: C.GoString(item.key), Value: C.GoString(item.value), AuthorUserId: C.GoString(item.authorUserId), Revision: uint64(item.revision), UpdateTs: uint64(item.updateTs)}
	}
	return dst
}

func CLockDetailToLockDetail(src *C.struct_C_LockDetail) *LockDetail {
	if src == nil {
		return nil
	}
	return &LockDetail{LockName: C.GoString(src.lockName), Owner: C.GoString(src.owner), Ttl: uint32(src.ttl)}
}

func copyLockDetails(ptr *C.struct_C_LockDetail, count C.size_t) []LockDetail {
	src := nativeSlice(ptr, count)
	dst := make([]LockDetail, len(src))
	for i := range src {
		dst[i] = *CLockDetailToLockDetail(&src[i])
	}
	return dst
}

func copyStateItems(ptr *C.struct_C_StateItem, count C.size_t) []StateItem {
	src := nativeSlice(ptr, count)
	dst := make([]StateItem, len(src))
	for i := range src {
		dst[i] = StateItem{Key: C.GoString(src[i].key), Value: C.GoString(src[i].value)}
	}
	return dst
}

func CUserStateToUserState(src *C.struct_C_UserState) *UserState {
	if src == nil {
		return nil
	}
	return &UserState{UserId: C.GoString(src.userId), States: copyStateItems(src.states, src.statesCount)}
}

func copyUserStates(ptr *C.struct_C_UserState, count C.size_t) []UserState {
	src := nativeSlice(ptr, count)
	dst := make([]UserState, len(src))
	for i := range src {
		dst[i] = *CUserStateToUserState(&src[i])
	}
	return dst
}

func CChannelInfoToChannelInfo(src *C.struct_C_ChannelInfo) *ChannelInfo {
	if src == nil {
		return nil
	}
	return &ChannelInfo{ChannelName: C.GoString(src.channelName), ChannelType: RtmChannelType(src.channelType)}
}

func copyChannelInfos(ptr *C.struct_C_ChannelInfo, count C.size_t) []ChannelInfo {
	src := nativeSlice(ptr, count)
	dst := make([]ChannelInfo, len(src))
	for i := range src {
		dst[i] = *CChannelInfoToChannelInfo(&src[i])
	}
	return dst
}

func CLinkStateEventToLinkStateEvent(src *C.struct_C_LinkStateEvent) *LinkStateEvent {
	if src == nil {
		return nil
	}
	return &LinkStateEvent{CurrentState: int(src.currentState), PreviousState: int(src.previousState), ServiceType: RtmServiceType(src.serviceType), Operation: int(src.operation), ReasonCode: int(src.reasonCode), Reason: C.GoString(src.reason), AffectedChannels: copyStrings(src.affectedChannels, src.affectedChannelCount), UnrestoredChannels: copyStrings(src.unrestoredChannels, src.unrestoredChannelCount), IsResumed: bool(src.isResumed), Timestamp: uint64(src.timestamp)}
}

func copyMessageBytes(ptr *C.char, count C.size_t) []byte {
	src := nativeSlice((*byte)(unsafe.Pointer(ptr)), count)
	return append([]byte(nil), src...)
}

func CHistoryMessageToHistoryMessage(src *C.struct_C_HistoryMessage) *HistoryMessage {
	if src == nil {
		return nil
	}
	message := string(copyMessageBytes(src.message, src.messageLength))
	return &HistoryMessage{MessageType: RtmMessageType(src.messageType), Message: message, MessageLength: uint(len(message)), Timestamp: uint64(src.timestamp), Publisher: C.GoString(src.publisher), CustomType: C.GoString(src.customType)}
}

func copyHistoryMessages(ptr *C.struct_C_HistoryMessage, count C.size_t) []HistoryMessage {
	src := nativeSlice(ptr, count)
	dst := make([]HistoryMessage, len(src))
	for i := range src {
		dst[i] = *CHistoryMessageToHistoryMessage(&src[i])
	}
	return dst
}

// newCStringArray allocates both the pointer array and strings in C memory.
func newCStringArray(values []string) (**C.char, func()) {
	if len(values) == 0 {
		return nil, func() {}
	}
	ptr := (**C.char)(C.calloc(C.size_t(len(values)), C.size_t(unsafe.Sizeof(uintptr(0)))))
	elems := unsafe.Slice(ptr, len(values))
	for i, value := range values {
		elems[i] = C.CString(value)
	}
	return ptr, func() {
		for _, value := range elems {
			C.free(unsafe.Pointer(value))
		}
		C.free(unsafe.Pointer(ptr))
	}
}

// newCMetadata returns a contiguous C array with all nested strings C-owned.
func newCMetadata(data *IMetadata) (*C.struct_C_Metadata, func()) {
	if data == nil {
		return nil, func() {}
	}
	ptr := C.C_Metadata_New()
	ptr.majorRevision = C.int64_t(data.majorRevision)
	ptr.itemCount = C.size_t(len(data.items))
	var elems []C.struct_C_MetadataItem
	if len(data.items) > 0 {
		ptr.items = (*C.struct_C_MetadataItem)(C.calloc(ptr.itemCount, C.size_t(C.sizeof_struct_C_MetadataItem)))
		elems = unsafe.Slice(ptr.items, len(data.items))
		for i, item := range data.items {
			elems[i].key = C.CString(item.Key)
			elems[i].value = C.CString(item.Value)
			elems[i].authorUserId = C.CString(item.AuthorUserId)
			elems[i].revision = C.int64_t(item.Revision)
			elems[i].updateTs = C.int64_t(item.UpdateTs)
		}
	}
	return ptr, func() {
		for i := range elems {
			C.free(unsafe.Pointer(elems[i].key))
			C.free(unsafe.Pointer(elems[i].value))
			C.free(unsafe.Pointer(elems[i].authorUserId))
		}
		C.free(unsafe.Pointer(ptr.items))
		C.C_Metadata_Delete(ptr)
	}
}

func newCStateItems(items []*StateItem) (*C.struct_C_StateItem, func()) {
	if len(items) == 0 {
		return nil, func() {}
	}
	ptr := (*C.struct_C_StateItem)(C.calloc(C.size_t(len(items)), C.size_t(C.sizeof_struct_C_StateItem)))
	elems := unsafe.Slice(ptr, len(items))
	for i, item := range items {
		elems[i].key = C.CString(item.Key)
		elems[i].value = C.CString(item.Value)
	}
	return ptr, func() {
		for i := range elems {
			C.free(unsafe.Pointer(elems[i].key))
			C.free(unsafe.Pointer(elems[i].value))
		}
		C.free(unsafe.Pointer(ptr))
	}
}
