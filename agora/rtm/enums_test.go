package agorartm

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestPinnedEnumCoverage(t *testing.T) {
	header, err := os.ReadFile("../headers/agora_rtm_sdk_c/include/C_AgoraRtmBase.h")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile("enums.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TOKEN_EVENT_TYPE", "LINK_STATE", "LINK_OPERATION", "LINK_STATE_CHANGE_REASON", "PROTOCOL_TYPE", "ERROR_CODE", "STORAGE_EVENT_TYPE", "LOCK_EVENT_TYPE", "TOPIC_EVENT_TYPE", "PRESENCE_EVENT_TYPE"} {
		body := regexp.MustCompile(`(?s)enum C_RTM_` + name + `\s*\{(.*?)\};`).FindSubmatch(header)
		if len(body) != 2 {
			t.Fatalf("missing enum %s", name)
		}
		for _, match := range regexp.MustCompile(`\b(RTM_[A-Z0-9_]+)\s*=`).FindAllSubmatch(body[1], -1) {
			if !strings.Contains(string(generated), "= C."+string(match[1])+"\n") {
				t.Errorf("missing native enum %s", match[1])
			}
		}
	}
}

func TestErrorCodeReasonAndMetadataAccessor(t *testing.T) {
	if RtmErrorCodeInvalidParameter != -10014 || RtmErrorCodeOK != 0 || RtmLinkStateConnected != 2 || RtmProtocolTypeTCPOnly != 1 {
		t.Fatal("native enum values drifted")
	}
	if RtmErrorCodeInvalidParameter.Reason() != GetErrorReason(-10014) || RtmErrorCodeInvalidParameter.Reason() == "" {
		t.Fatal("reason does not preserve native signed code")
	}
	item := MetadataItem{AuthorUserId: "author"}
	if item.GetAuthorUserId() != "author" || item.GetAquthorUserId() != item.GetAuthorUserId() {
		t.Fatal("metadata author accessor")
	}
}
