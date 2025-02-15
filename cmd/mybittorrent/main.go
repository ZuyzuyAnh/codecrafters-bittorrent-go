package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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

func (t TorrentFile) hashInfoByte() ([]byte, error) {
	bencodedInfo := bencode.BencodeMap(t.Info)

	hasher := sha1.New()
	hasher.Write([]byte(bencodedInfo))
	hashed := hasher.Sum(nil)

	return hashed, nil
}

func (t TorrentFile) hashPieces() ([]string, error) {
	pieces := t.Info["pieces"].([]byte)
	res := make([]string, 0)

	for i := 0; i < len(pieces); i += 20 {
		hashed := pieces[i : i+20]
		res = append(res, hex.EncodeToString(hashed))
	}

	return res, nil
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

func (t TorrentFile) buildTrackerRequest() (string, error) {
	trackerUrl := t.Announce
	infoHash, err := t.hashInfoByte()
	if err != nil {
		return "", err
	}

	params := url.Values{}

	params.Add("info_hash", string(infoHash))
	params.Add("peer_id", "-PC0001-"+"123456789012")
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprint(t.Info["length"]))
	params.Add("compact", "1")

	return trackerUrl + "?" + params.Encode(), nil
}

func (t TorrentFile) sendGetRequest() (string, error) {
	url, err := t.buildTrackerRequest()
	if err != nil {
		return "", err
	}

	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	decoded, _, err := decode.DecodeBencode(string(body))
	if err != nil {
		return "", err
	}

	return decoded.(map[string]interface{})["peers"].(string), nil
}

func parsePeers(peersStr string) ([]string, error) {
	peersBytes := []byte(peersStr)

	if len(peersBytes)%6 != 0 {
		return nil, fmt.Errorf("invalid peers length: %d", len(peersBytes))
	}

	var peers []string
	for i := 0; i < len(peersBytes); i += 6 {
		ip := fmt.Sprintf("%d.%d.%d.%d",
			peersBytes[i],
			peersBytes[i+1],
			peersBytes[i+2],
			peersBytes[i+3])

		port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])
		peers = append(peers, fmt.Sprintf("%s:%d", ip, port))
	}
	return peers, nil
}

func (t TorrentFile) performHandShake(peerAddr string) (string, error) {
	const protocol = "BitTorrent protocol"
	const protocolLen = 19
	reserved := make([]byte, 8)

	infoHash, err := t.hashInfoByte()
	if err != nil {
		return "", err
	}

	peerID := make([]byte, 20)
	_, err = rand.Read(peerID)
	if err != nil {
		return "", err
	}

	handshake := make([]byte, 0, 68)
	handshake = append(handshake, byte(protocolLen))
	handshake = append(handshake, []byte(protocol)...)
	handshake = append(handshake, reserved...)
	handshake = append(handshake, infoHash...)
	handshake = append(handshake, peerID...)

	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = conn.Write(handshake)
	if err != nil {
		return "", err
	}

	response := make([]byte, 68)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return "", err
	}

	peerId := response[48:68]
	return string(peerId), nil
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
		fmt.Println("Piece Length:", torrentFile.Info["piece length"])

		pieces, err := torrentFile.hashPieces()
		fmt.Println("Pieces:")
		for _, piece := range pieces {
			fmt.Println(piece)
		}
	case "peers":
		file := os.Args[2]

		torrentFile, err := infoFile(file)
		if err != nil {
			fmt.Println(err)
			return
		}

		peers, err := torrentFile.sendGetRequest()
		if err != nil {
			fmt.Println(err)
			return
		}

		parsedPeers, err := parsePeers(peers)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, peer := range parsedPeers {
			fmt.Println(peer)
		}
	case "handshake":
		file := os.Args[2]
		peerAddr := os.Args[3]

		torrentFile, err := infoFile(file)
		if err != nil {
			fmt.Println(err)
			return
		}

		peerID, err := torrentFile.performHandShake(peerAddr)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Peer ID:", peerID)

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
