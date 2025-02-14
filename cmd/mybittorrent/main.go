package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

func getFirstColonIndex(bencodedString string) int {
	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			return i
		}
	}

	return -1
}

func decodeBenCodeString(firstColonIndex int, bencodedString string) (string, error) {
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

func lastIndexInteger(start int, bencodedString string) int {
	for i := start; i < len(bencodedString); i++ {
		if bencodedString[i] == 'e' {
			return i
		}
	}

	return -1
}

func decodeBencodeList(bencodedString string) ([]interface{}, error) {
	pointer := 1
	decodedList := make([]interface{}, 0)

	for pointer < len(bencodedString) {
		if unicode.IsDigit(rune(bencodedString[0])) {
			firstColonIndex := getFirstColonIndex(bencodedString)
			decodedString, err := decodeBenCodeString(firstColonIndex, bencodedString)
			if err != nil {
				return nil, err
			}

			decodedList = append(decodedList, decodedString)

			pointer += len(decodedString) + 1
		} else if bencodedString[pointer] == 'i' {
			lastIndexInteger := lastIndexInteger(pointer, bencodedString)
			decodedInt, err := decodeBencodeInt(bencodedString[pointer : lastIndexInteger+1])
			if err != nil {
				return nil, err
			}

			decodedList = append(decodedList, decodedInt)
			pointer += lastIndexInteger + 1
		}
	}

	return decodedList, nil
}

func decodeBencode(bencodedString string) (interface{}, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		firstColonIndex := getFirstColonIndex(bencodedString)
		return decodeBenCodeString(firstColonIndex, bencodedString)
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
