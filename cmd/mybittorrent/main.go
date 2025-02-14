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
	pointer := 1
	decodedList := make([]interface{}, 0)

	for pointer < len(bencodedString) {
		if bencodedString[pointer] == 'e' {
			pointer++
			break
		}

		decoded, consumed, err := decodeBencode(bencodedString[pointer:])
		if err != nil {
			return nil, 0, err
		}
		decodedList = append(decodedList, decoded)
		pointer += consumed
	}

	return decodedList, pointer, nil
}

func decodeBencodeDict(bencodedString string) (map[string]interface{}, int, error) {
	pointer := 1
	decodedDict := make(map[string]interface{})

	for pointer < len(bencodedString) {
		if bencodedString[pointer] == 'e' {
			pointer++
			break
		}

		key, consumed, err := decodeBenCodeString(bencodedString[pointer:])
		if err != nil {
			return nil, 0, err
		}
		pointer += consumed

		decoded, consumed, err := decodeBencode(bencodedString[pointer:])
		if err != nil {
			return nil, 0, err
		}
		decodedDict[key] = decoded
		pointer += consumed
	}

	return decodedDict, pointer, nil
}

func decodeBencode(bencodedString string) (interface{}, int, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		return decodeBenCodeString(bencodedString)
	}

	if bencodedString[0] == 'i' && bencodedString[len(bencodedString)-1] == 'e' {
		return decodeBencodeInt(bencodedString)
	}

	if bencodedString[0] == 'l' {
		return decodeBencodeList(bencodedString)
	}

	if bencodedString[0] == 'd' {
		return decodeBencodeDict(bencodedString)
	}

	return "", 0, fmt.Errorf("only strings are supported at the moment")
}

type Info struct {
	Length      int    `json:"length"`
	Name        string `json:"name"`
	PieceLength int    `json:"piece length"`
	Pieces      []byte `json:"pieces"`
}

type TorrentFile struct {
	Announce string `json:"announce"`
	Info     Info   `json:"info"`
}

func infoFile(file string) (TorrentFile, error) {
	f, err := os.Open(file)
	if err != nil {
		fmt.Println("cannot open file", err)
		return TorrentFile{}, err
	}
	defer f.Close()

	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return TorrentFile{}, err
	}

	decodedValue, _, err := decodeBencode(string(data))
	if err != nil {
		fmt.Println("Error decoding:", err)
		return TorrentFile{}, err
	}

	decoded := decodedValue.(map[string]interface{})
	torrentFile := TorrentFile{
		Announce: decoded["announce"].(string),
		Info: Info{
			Length:      decoded["info"].(map[string]interface{})["length"].(int),
			Name:        decoded["info"].(map[string]interface{})["name"].(string),
			PieceLength: decoded["info"].(map[string]interface{})["piece length"].(int),
			Pieces:      []byte(decoded["info"].(map[string]interface{})["pieces"].(string)),
		},
	}

	return torrentFile, nil
}

func jsonOutput(data interface{}) {
	jsonOutput, _ := json.Marshal(data)
	fmt.Println(string(jsonOutput))
}

func main() {
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]

		decoded, _, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput(decoded)
	case "info":
		file := os.Args[2]

		torrentFile, err := infoFile(file)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput(torrentFile)
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
