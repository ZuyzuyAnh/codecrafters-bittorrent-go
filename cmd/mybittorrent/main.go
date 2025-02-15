package main

import (
	"bytes"
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
	"strconv"

	bencode "github.com/codecrafters-io/bittorrent-starter-go/internal/bencoder"
	decode "github.com/codecrafters-io/bittorrent-starter-go/internal/decoder"
)

const (
	msgChoke      = 0
	msgUnchoke    = 1
	msgInterested = 2
	msgBitfield   = 5
	msgRequest    = 6
	msgPiece      = 7
)

type TorrentFile struct {
	Announce string                 `json:"Tracker URL"`
	Info     map[string]interface{} `json:"info"`
}

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
	return hasher.Sum(nil), nil
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

func infoFile(file string) (TorrentFile, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return TorrentFile{}, err
	}

	decodedValue, _, err := decode.DecodeBencode(string(data))
	if err != nil {
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

func (t TorrentFile) performHandShake(conn net.Conn) (string, error) {
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
	return hex.EncodeToString(peerId), nil
}

func sendMessage(conn net.Conn, id byte, payload []byte) error {
	length := uint32(1 + len(payload))
	buffer := make([]byte, 4+1+len(payload))
	binary.BigEndian.PutUint32(buffer[:4], length)
	buffer[4] = id
	copy(buffer[5:], payload)
	_, err := conn.Write(buffer)
	return err
}

func receiveMessage(conn net.Conn) (byte, []byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	length := binary.BigEndian.Uint32(header[:4])
	if length == 0 {
		return 0, nil, nil
	}

	msg := make([]byte, length)
	if _, err := io.ReadFull(conn, msg); err != nil {
		return 0, nil, err
	}
	id := msg[0]
	payload := msg[1:]
	return id, payload, nil
}

func downloadPiece(conn net.Conn, pieceIndex int, pieceLength int) ([]byte, error) {
	blockSize := 16 * 1024
	numBlocks := pieceLength / blockSize
	if pieceLength%blockSize != 0 {
		numBlocks++
	}

	pieceBuf := make([]byte, pieceLength)

	for i := 0; i < numBlocks; i++ {
		begin := i * blockSize
		reqLen := blockSize

		if begin+reqLen > pieceLength {
			reqLen = pieceLength - begin
		}

		payload := make([]byte, 12)

		binary.BigEndian.PutUint32(payload[:4], uint32(pieceIndex))
		binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
		binary.BigEndian.PutUint32(payload[8:12], uint32(reqLen))

		if err := sendMessage(conn, msgRequest, payload); err != nil {
			return nil, fmt.Errorf("error sending request: %v", err)
		}
	}

	receivedBytes := 0
	for receivedBytes < pieceLength {
		id, payload, err := receiveMessage(conn)
		if err != nil {
			return nil, fmt.Errorf("error receiving message: %v", err)
		}
		if id != msgPiece {
			continue
		}
		if len(payload) < 8 {
			return nil, fmt.Errorf("invalid piece message length")
		}

		begin := binary.BigEndian.Uint32(payload[4:8])
		block := payload[8:]
		copy(pieceBuf[begin:], block)
		receivedBytes += len(block)
	}

	return pieceBuf, nil
}

func jsonOutput(data interface{}) {
	out, _ := json.Marshal(data)
	fmt.Println(string(out))
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
		if err != nil {
			fmt.Println("Error getting pieces:", err)
			return
		}
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
		conn, err := net.Dial("tcp", peerAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()
		peerID, err := torrentFile.performHandShake(conn)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Peer ID:", peerID)
	case "download_piece":
		if len(os.Args) < 6 {
			fmt.Println("Usage: download_piece -o <output_path> <torrent_file> <piece_index>")
			return
		}
		outputPath := os.Args[3]
		torrentPath := os.Args[4]
		pieceIndexStr := os.Args[5]
		pieceIndex, err := strconv.Atoi(pieceIndexStr)
		if err != nil {
			fmt.Println("Invalid piece index:", err)
			return
		}

		torrentFile, err := infoFile(torrentPath)
		if err != nil {
			fmt.Println(err)
			return
		}

		peersStr, err := torrentFile.sendGetRequest()
		if err != nil {
			fmt.Println("Error getting peers:", err)
			return
		}
		parsedPeers, err := parsePeers(peersStr)
		if err != nil {
			fmt.Println("Error parsing peers:", err)
			return
		}
		if len(parsedPeers) == 0 {
			fmt.Println("No peers found")
			return
		}
		chosenPeer := parsedPeers[0]
		conn, err := net.Dial("tcp", chosenPeer)
		if err != nil {
			fmt.Println("Error connecting to peer:", err)
			return
		}
		defer conn.Close()

		_, err = torrentFile.performHandShake(conn)
		if err != nil {
			fmt.Println("Handshake error:", err)
			return
		}
		for {
			id, _, err := receiveMessage(conn)
			if err != nil {
				fmt.Println("Error receiving bitfield:", err)
				return
			}
			if id == msgBitfield {
				break
			}
		}

		if err := sendMessage(conn, msgInterested, []byte{}); err != nil {
			fmt.Println("Error sending interested:", err)
			return
		}

		for {
			id, _, err := receiveMessage(conn)
			if err != nil {
				fmt.Println("Error waiting for unchoke:", err)
				return
			}
			if id == msgUnchoke {
				break
			}
		}

		standardPieceLength := torrentFile.Info["piece length"].(int)
		fileLength := torrentFile.Info["length"].(int)

		numPieces := fileLength / standardPieceLength
		if fileLength%standardPieceLength != 0 {
			numPieces++
		}

		var actualPieceLength int
		if pieceIndex == numPieces-1 {
			actualPieceLength = fileLength - (numPieces-1)*standardPieceLength

		} else {
			actualPieceLength = standardPieceLength
		}

		pieceData, err := downloadPiece(conn, pieceIndex, actualPieceLength)
		if err != nil {
			fmt.Println("Error downloading piece:", err)
			return
		}

		sha1sum := sha1.Sum(pieceData)
		fmt.Printf("Downloaded piece SHA-1: %x\n", sha1sum)

		err = os.WriteFile(outputPath, pieceData, 0644)
		if err != nil {
			fmt.Println("Error writing piece to disk:", err)
			return
		}
		fmt.Printf("Piece %d downloaded and saved to %s\n", pieceIndex, outputPath)
	case "download":
		if len(os.Args) < 5 {
			fmt.Println("Usage: download -o <torrent_file> <output_dir>")
			return
		}

		outputPath := os.Args[3]
		torrentPath := os.Args[4]

		torrentFile, err := infoFile(torrentPath)
		if err != nil {
			fmt.Println("error reading torrent file:", err)
			return
		}

		fileLength := torrentFile.Info["length"].(int)
		standardPieceLength := torrentFile.Info["piece length"].(int)

		numPieces := fileLength / standardPieceLength

		if fileLength%standardPieceLength != 0 {
			numPieces++
		}

		pieces := torrentFile.Info["pieces"].([]byte)
		expectedHash := make([][]byte, 0, numPieces)

		for i := 0; i < len(pieces); i += 20 {
			expectedHash = append(expectedHash, pieces[i:i+20])
		}

		peersStr, err := torrentFile.sendGetRequest()
		if err != nil {
			fmt.Println("error getting peers:", err)
			return
		}

		parsedPeers, err := parsePeers(peersStr)
		if err != nil {
			fmt.Println("error parsing peers:", err)
			return
		}

		if len(parsedPeers) == 0 {
			fmt.Println("no peers found")
			return
		}

		chosenPeer := parsedPeers[0]

		conn, err := net.Dial("tcp", chosenPeer)
		if err != nil {
			fmt.Println("error connecting to peer:", err)
			return
		}
		defer conn.Close()

		if _, err := torrentFile.performHandShake(conn); err != nil {
			fmt.Println("error performing handshake:", err)
			return
		}

		// Stage 1: Receive bitfield
		for {
			id, _, err := receiveMessage(conn)
			if err != nil {
				fmt.Println("error receiving bitfield:", err)
				return
			}
			if id == msgBitfield {
				break
			}
		}

		// Stage 2: Send interested
		if err := sendMessage(conn, msgInterested, []byte{}); err != nil {
			fmt.Println("error sending interested:", err)
			return
		}

		// Stage 3: Receive unchoke
		for {
			id, _, err := receiveMessage(conn)
			if err != nil {
				fmt.Println("error waiting for unchoke:", err)
				return
			}
			if id == msgUnchoke {
				break
			}
		}

		// Stage 4: Download pieces
		fileBuffer := make([]byte, fileLength)
		offset := 0

		for i := 0; i < numPieces; i++ {
			var actualPieceLength int
			if i < numPieces-1 {
				actualPieceLength = standardPieceLength
			} else {
				actualPieceLength = fileLength - (numPieces-1)*standardPieceLength
			}

			pieceData, err := downloadPiece(conn, i, actualPieceLength)
			if err != nil {
				fmt.Println("error downloading piece:", err)
				return
			}

			computedHash := sha1.Sum(pieceData)
			if !bytes.Equal(computedHash[:], expectedHash[i]) {
				fmt.Printf("piece %d hash mismatch\n", i)
				return
			}

			copy(fileBuffer[offset:], pieceData)
			offset += actualPieceLength
		}

		err = os.WriteFile(outputPath, fileBuffer, 0644)
		if err != nil {
			fmt.Println("error writing file to disk:", err)
			return
		}

		fmt.Printf("file downloaded and saved to %s\n", outputPath)
	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}
