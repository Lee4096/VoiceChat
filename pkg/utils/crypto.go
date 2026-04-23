package utils

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"regexp"
)

func MD5(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func Base64Decode(str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(str)
}

func Int16ToBytes(data []int16) []byte {
	bytes := make([]byte, len(data)*2)
	for i, v := range data {
		binary.LittleEndian.PutUint16(bytes[i*2:], uint16(v))
	}
	return bytes
}

func BytesToInt16(data []byte) []int16 {
	if len(data)%2 != 0 {
		data = data[:len(data)/2*2]
	}
	result := make([]int16, len(data)/2)
	for i := range result {
		result[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}
	return result
}

var jsonEscapeRegex = regexp.MustCompile(`["\\]`)

func EscapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	s = string(b)
	return s[1 : len(s)-1]
}
