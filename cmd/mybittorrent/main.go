package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	bencode "github.com/codecrafters-io/bittorrent-starter-go/internal/bencoder"
	decode "github.com/codecrafters-io/bittorrent-starter-go/internal/decoder"
)

func (t TorrentFile) hashInfo() (string, error) {
	bencodedInfo := bencode.BencodeMap(t.Info)

	hasher := sha1.New()
	hasher.Write([]byte(bencodedInfo))
	hashed := hasher.Sum(nil)

	return hex.EncodeToString(hashed), nil
}

type TorrentFile struct {
	Announce string                 `json:"Tracker URL"`
	Info     map[string]interface{} `json:"info"`
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

	decodedValue, _, err := decode.DecodeBencode(string(data))
	if err != nil {
		fmt.Println("Error decoding:", err)
		return TorrentFile{}, err
	}

	decoded := decodedValue.(map[string]interface{})
	torrentFile := TorrentFile{
		Announce: decoded["announce"].(string),
		Info: map[string]interface{}{
			"length":       decoded["info"].(map[string]interface{})["length"].(int),
			"name":         decoded["info"].(map[string]interface{})["name"].(string),
			"piece length": decoded["info"].(map[string]interface{})["piece length"].(int),
			"pieces":       []byte(decoded["info"].(map[string]interface{})["pieces"].(string)),
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

		decoded, _, err := decode.DecodeBencode(bencodedValue)
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

		fmt.Println("Tracker URL:", torrentFile.Announce)
		fmt.Println("Length:", torrentFile.Info["length"])

		infoHash, err := torrentFile.hashInfo()
		if err != nil {
			fmt.Println("Error calculating info hash:", err)
			return
		}

		fmt.Println("Info Hash:", infoHash)
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
