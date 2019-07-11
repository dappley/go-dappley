package network

import (
	"errors"
	"github.com/libp2p/go-libp2p-core/peer"
	"math/big"
	"reflect"
)

const (
	lengthByteLength     = 8
	startByteLength      = 2
	checkSumLength       = 1
	headerCheckSumLength = 1
	headerLength         = startByteLength + lengthByteLength + checkSumLength + headerCheckSumLength

	startBytesIndex         = 0
	lengthBytesIndex        = startBytesIndex + startByteLength
	checkSumByteIndex       = lengthBytesIndex + lengthByteLength
	headerCheckSumByteIndex = checkSumByteIndex + checkSumLength
)

var (
	startBytes = []byte{0x7E, 0x7E}
)

var (
	ErrLengthTooShort       = errors.New("message length is too short")
	ErrInvalidMessageFormat = errors.New("invalid message format")
	ErrCheckSumIncorrect    = errors.New("incorrect checksum")
)

type DappPacket struct {
	header []byte
	data   []byte
}

type DappPacketContext struct {
	packet *DappPacket
	source peer.ID
}

func ConstructDappPacketFromData(data []byte) *DappPacket {
	packet := &DappPacket{}

	packet.header = constructHeader(data)
	packet.data = data
	return packet
}

func ExtractDappPacketFromRawBytes(bytes []byte) (*DappPacket, error) {
	packet := &DappPacket{}

	if len(bytes) <= headerLength {
		return nil, ErrLengthTooShort
	}

	packet.header = bytes[:headerLength]
	err := packet.verifyHeader()

	if err != nil {
		return nil, err
	}

	if len(bytes) < headerLength+packet.GetPacketDataLength() {
		return nil, ErrLengthTooShort
	}

	packet.data = bytes[headerLength : headerLength+packet.GetPacketDataLength()]

	err = packet.verifyDataChecksum()

	if err != nil {
		return nil, err
	}

	return packet, nil
}

func (packet *DappPacket) GetHeader() []byte { return packet.header }
func (packet *DappPacket) GetData() []byte   { return packet.data }
func (packet *DappPacket) GetStartBytes() []byte {
	return packet.header[startBytesIndex : startBytesIndex+len(startBytes)]
}
func (packet *DappPacket) GetLengthBytes() []byte {
	return packet.header[lengthBytesIndex : lengthBytesIndex+lengthByteLength]
}
func (packet *DappPacket) GetCheckSum() byte       { return packet.header[checkSumByteIndex] }
func (packet *DappPacket) GetHeaderCheckSum() byte { return packet.header[headerCheckSumByteIndex] }
func (packet *DappPacket) GetPacketDataLength() int {
	l := *new(big.Int).SetBytes(packet.GetLengthBytes())
	return int(l.Uint64())
}

func (packet *DappPacket) GetLength() int {
	return len(packet.header) + len(packet.data)
}

func (packet *DappPacket) GetRawBytes() []byte {
	return append(packet.header, packet.data...)
}

func (packet *DappPacket) verifyHeader() error {
	if len(packet.header) != headerLength {
		return ErrLengthTooShort
	}

	if !packet.containStartingBytes() {
		return ErrInvalidMessageFormat
	}

	headerCheckSum := checkSum(packet.header[:headerLength-1])
	if headerCheckSum != packet.GetHeaderCheckSum() {
		return ErrCheckSumIncorrect
	}

	return nil
}

func (packet *DappPacket) verifyDataChecksum() error {
	dataCheckSum := checkSum(packet.data)

	if dataCheckSum != packet.GetCheckSum() {
		return ErrCheckSumIncorrect
	}

	return nil
}

func (packet *DappPacket) containStartingBytes() bool {
	if len(packet.header) < startByteLength {
		return false
	}

	return reflect.DeepEqual(packet.GetStartBytes(), startBytes)
}

func constructHeader(data []byte) []byte {

	length := len(data)
	msg := make([]byte, lengthByteLength)
	lengthBytes := big.NewInt(int64(length)).Bytes()
	lenDiff := len(msg) - len(lengthBytes)
	for i, b := range lengthBytes {
		msg[i+lenDiff] = b
	}

	header := append(startBytes, msg...)
	cs := checkSum(data)
	header = append(header, cs)
	headerCs := checkSum(header)
	header = append(header, headerCs)
	return header
}

func checkSum(data []byte) byte {
	sum := byte(0)
	for _, d := range data {
		sum += d
	}
	return sum
}
