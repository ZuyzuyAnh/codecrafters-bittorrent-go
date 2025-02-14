package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

func decodeBenCodeString(bencodedString string) (string, error) {
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
		return "", err
	}

	return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
}

func decodeBencodeInt(bencodedString string) (int, error) {
	return strconv.Atoi(bencodedString[1 : len(bencodedString)-1])
}

func decodeBencodeList(bencodedString string) ([]interface{}, error) {
	pointer := 0
	decodedList := make([]interface{}, 0)

	for {
		if pointer >= len(bencodedString) {
			break
		}

		if bencodedString[pointer] == 'e' {
			break
		}

		decoded, err := decodeBencode(bencodedString[pointer:])
		if err != nil {
			return nil, err
		}

		decodedList = append(decodedList, decoded)
	}

	return decodedList, nil
}

func decodeBencode(bencodedString string) (interface{}, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		return decodeBenCodeString(bencodedString)
	}

	if bencodedString[0] == 'i' && bencodedString[len(bencodedString)-1] == 'e' {
		return decodeBencodeInt(bencodedString)
	}

	if bencodedString[0] == 'l' && bencodedString[len(bencodedString)-1] == 'e' {
		return decodeBencodeList(bencodedString[1 : len(bencodedString)-1])
	}

	return "", fmt.Errorf("only strings are supported at the moment")
}

func main() {
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
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
