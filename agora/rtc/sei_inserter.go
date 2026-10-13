package agoraservice

import "bytes"

// AddSEI creates an Agora payload-type 0x65 SEI NAL for H.264 or H.265.
func AddSEI(seiMsg []byte, isH264 bool) []byte {
	overhead := len(seiMsg)/255 + 10
	if len(seiMsg) > int(^uint(0)>>1)-overhead {
		return nil
	}
	result := make([]byte, 0, len(seiMsg)+overhead)
	result = append(result, 0, 0, 0, 1)
	if isH264 {
		result = append(result, 0x06)
	} else {
		result = append(result, 0x4e, 0x01)
	}
	result = append(result, 0x65)
	remaining := len(seiMsg)
	for remaining >= 255 {
		result = append(result, 0xff)
		remaining -= 255
	}
	result = append(result, byte(remaining))
	result = append(result, seiMsg...)
	return append(result, 0x80)
}

func is264NaluHeader(data []byte, index int) bool {
	return index >= 0 && index < len(data)-1 && data[index]&0x1f == 0x06 && data[index+1] == 0x65
}

func is265NaluHeader(data []byte, index int) bool {
	if index < 0 || index >= len(data)-2 {
		return false
	}
	kind := (data[index] & 0x7e) >> 1
	return (kind == 39 || kind == 40) && data[index+2] == 0x65
}

// FindSEI returns a copied Agora SEI payload and its size, or a negative code.
// It accepts three- and four-byte Annex B start codes and rejects truncated data.
func FindSEI(encoded []byte, codecType VideoCodecType) ([]byte, int) {
	if len(encoded) == 0 {
		return nil, -1000
	}
	if codecType != VideoCodecTypeH264 && codecType != VideoCodecTypeH265 {
		return nil, -1003
	}
	if len(encoded) < 6 {
		return nil, -1001
	}
	startCode := []byte{0, 0, 1}
	for search := 0; search < len(encoded); {
		start := bytes.Index(encoded[search:], startCode)
		if start < 0 {
			break
		}
		header := search + start + len(startCode)
		search = header
		offset := 0
		if codecType == VideoCodecTypeH264 && is264NaluHeader(encoded, header) {
			offset = 2
		}
		if codecType == VideoCodecTypeH265 && is265NaluHeader(encoded, header) {
			offset = 3
		}
		if offset == 0 {
			continue
		}
		index, size := header+offset, 0
		for {
			if index >= len(encoded) {
				return nil, -1002
			}
			part := int(encoded[index])
			index++
			if part > len(encoded)-size {
				return nil, -1002
			}
			size += part
			if part != 255 {
				break
			}
		}
		if size >= len(encoded)-index || encoded[index+size] != 0x80 {
			return nil, -1002
		}
		payload := make([]byte, size)
		copy(payload, encoded[index:index+size])
		return payload, size
	}
	return nil, -1004
}

// InsertSEIToEncodedData returns a new buffer and never mutates encodedData.
func InsertSEIToEncodedData(encodedData []byte, sei []byte, codecType VideoCodecType) []byte {
	if codecType != VideoCodecTypeH264 && codecType != VideoCodecTypeH265 {
		return nil
	}
	if len(sei) == 0 {
		return bytes.Clone(encodedData)
	}
	nal := AddSEI(sei, codecType == VideoCodecTypeH264)
	if nal == nil || len(encodedData) > int(^uint(0)>>1)-len(nal) {
		return nil
	}
	result := make([]byte, 0, len(encodedData)+len(nal))
	result = append(result, encodedData...)
	return append(result, nal...)
}
