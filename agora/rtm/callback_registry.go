package agorartm

/*
#include "C_IAgoraRtmClient.h"
*/
import "C"
import "sync"

// Native code retains only its own C event handler pointer. Go objects remain in
// this synchronized registry; no pointer to Go pointer-containing memory crosses
// the cgo boundary. Entries are removed before native teardown. A lookup already
// in flight retains the immutable Go handler safely until its callback returns.
var rtmClients = struct {
	sync.RWMutex
	clients map[*C.struct_C_IRtmEventHandler]*IRtmClient
}{clients: make(map[*C.struct_C_IRtmEventHandler]*IRtmClient)}

func registerRtmClient(client *IRtmClient) {
	if client == nil || client.cEventHandler == nil {
		return
	}
	rtmClients.Lock()
	rtmClients.clients[client.cEventHandler] = client
	rtmClients.Unlock()
}

func unregisterRtmClient(client *IRtmClient) {
	if client == nil || client.cEventHandler == nil {
		return
	}
	rtmClients.Lock()
	delete(rtmClients.clients, client.cEventHandler)
	rtmClients.Unlock()
}

func lookupRtmClient(handler *C.struct_C_IRtmEventHandler) *IRtmClient {
	if handler == nil {
		return nil
	}
	rtmClients.RLock()
	client := rtmClients.clients[handler]
	rtmClients.RUnlock()
	return client
}
