package decode

import (
	"fmt"
	"strconv"
	"unicode"
)

func DecodeBenCodeString(bencodedString string) (string, int, error) {
	var firstColonIndex int

	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bencodedString[:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", 0, err
	}
	consumed := firstColonIndex + 1 + length
	return bencodedString[firstColonIndex+1 : consumed], consumed, nil
}

func DecodeBencodeInt(bencodedString string) (int, int, error) {
	var closingIndex int
	for i := 1; i < len(bencodedString); i++ {
		if bencodedString[i] == 'e' {
			closingIndex = i
			break
		}
	}

	val, err := strconv.Atoi(bencodedString[1:closingIndex])
	return val, closingIndex + 1, err
}

func DecodeBencodeList(bencodedString string) ([]interface{}, int, error) {
	pointer := 1
	decodedList := make([]interface{}, 0)

	for pointer < len(bencodedString) {
		if bencodedString[pointer] == 'e' {
			pointer++
			break
		}

		decoded, consumed, err := DecodeBencode(bencodedString[pointer:])
		if err != nil {
			return nil, 0, err
		}
		decodedList = append(decodedList, decoded)
		pointer += consumed
	}

	return decodedList, pointer, nil
}

func DecodeBencodeDict(bencodedString string) (map[string]interface{}, int, error) {
	pointer := 1
	decodedDict := make(map[string]interface{})

	for pointer < len(bencodedString) {
		if bencodedString[pointer] == 'e' {
			pointer++
			break
		}

		key, consumed, err := DecodeBenCodeString(bencodedString[pointer:])
		if err != nil {
			return nil, 0, err
		}
		pointer += consumed

		decoded, consumed, err := DecodeBencode(bencodedString[pointer:])
		if err != nil {
			return nil, 0, err
		}
		decodedDict[key] = decoded
		pointer += consumed
	}

	return decodedDict, pointer, nil
}

func DecodeBencode(bencodedString string) (interface{}, int, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		return DecodeBenCodeString(bencodedString)
	}

	if bencodedString[0] == 'i' && bencodedString[len(bencodedString)-1] == 'e' {
		return DecodeBencodeInt(bencodedString)
	}

	if bencodedString[0] == 'l' {
		return DecodeBencodeList(bencodedString)
	}

	if bencodedString[0] == 'd' {
		return DecodeBencodeDict(bencodedString)
	}

	return "", 0, fmt.Errorf("only strings are supported at the moment")
}
