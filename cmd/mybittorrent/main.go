package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

func decodeBenCodeString(bencodedString string) (string, int, error) {
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

func decodeBencodeInt(bencodedString string) (int, int, error) {
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

func decodeBencodeList(bencodedString string) ([]interface{}, int, error) {
	pointer := 0
	decodedList := make([]interface{}, 0)

	for {
		if pointer >= len(bencodedString) {
			break
		}

		if bencodedString[pointer] == 'e' {
			break
		}

		decoded, decodedLength, err := decodeBencode(bencodedString[pointer:])
		if err != nil {
			return nil, 0, err
		}

		pointer += decodedLength

		decodedList = append(decodedList, decoded)
	}

	return decodedList, len(bencodedString), nil
}

func decodeBencode(bencodedString string) (interface{}, int, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		return decodeBenCodeString(bencodedString)
	}

	if bencodedString[0] == 'i' && bencodedString[len(bencodedString)-1] == 'e' {
		return decodeBencodeInt(bencodedString)
	}

	if bencodedString[0] == 'l' && bencodedString[len(bencodedString)-1] == 'e' {
		return decodeBencodeList(bencodedString[1 : len(bencodedString)-1])
	}

	return "", 0, fmt.Errorf("only strings are supported at the moment")
}

func main() {
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, _, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
