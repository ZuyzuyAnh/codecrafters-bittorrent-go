package bencode

import (
	"bytes"
	"reflect"
	"sort"
	"strconv"
)

func BencodeMap(m map[string]interface{}) string {
	result := "d"
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		result += BencodeString(k)
		result += BencodeValue(m[k])
	}
	result += "e"
	return result
}

func BencodeString(s string) string {
	return strconv.Itoa(len(s)) + ":" + s
}

func BencodeInt(i int) string {
	return "i" + strconv.Itoa(i) + "e"
}

func BencodeList(l []interface{}) string {
	result := "l"
	for _, v := range l {
		result += BencodeValue(v)
	}
	result += "e"
	return result
}

func BencodeValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return BencodeString(val)
	case int:
		return "i" + strconv.Itoa(val) + "e"
	case map[string]interface{}:
		return BencodeMap(val)
	case []interface{}:
		return BencodeList(val)
	case []byte:
		var buf bytes.Buffer
		buf.WriteString(strconv.Itoa(len(val)))
		buf.WriteString(":")
		buf.Write(val)
		return buf.String()
	default:
		panic("not supported type: " + reflect.TypeOf(v).String())
	}
}
