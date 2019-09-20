package networkmodel

import (
	"errors"
	"math/big"
	"reflect"
)

const (
	startByteLength       = 2
	lengthByteLength      = 8
	isBroadcastByteLength = 1
	checkSumLength        = 1
	headerCheckSumLength  = 1
	headerLength          = startByteLength + lengthByteLength + isBroadcastByteLength + checkSumLength + headerCheckSumLength

	startBytesIndex         = 0
	lengthBytesIndex        = startBytesIndex + startByteLength
	isBroadcastByteIndex    = lengthBytesIndex + lengthByteLength
	checkSumByteIndex       = isBroadcastByteIndex + isBroadcastByteLength
	headerCheckSumByteIndex = checkSumByteIndex + checkSumLength
)

var (
	startBytes    = []byte{0x7E, 0x7E}
	broadcastByte = byte(1)
	unitcastByte  = byte(0)
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
	Packet *DappPacket
	Source PeerInfo
}

//ConstructDappPacketFromData creates a header for the input content and returns a DappPacket
func ConstructDappPacketFromData(data []byte, isBroadcast bool) *DappPacket {
	packet := &DappPacket{}

	packet.header = constructHeader(data, isBroadcast)
	packet.data = data
	return packet
}

//DeserializeIntoDappPacket deserializes raw bytes into DappPacket
func DeserializeIntoDappPacket(bytes []byte) (*DappPacket, error) {
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

//GetHeader returns the header bytes
func (packet *DappPacket) GetHeader() []byte { return packet.header }

//GetData returns the data bytes
func (packet *DappPacket) GetData() []byte { return packet.data }

//GetStartBytes returns the start bytes in the header
func (packet *DappPacket) GetStartBytes() []byte {
	return packet.header[startBytesIndex : startBytesIndex+len(startBytes)]
}

//GetBroadcastByte returns the broadcast byte in the header
func (packet *DappPacket) GetBroadcastByte() byte {
	return packet.header[isBroadcastByteIndex]
}

//GetLengthBytes returns the length byte in the header
func (packet *DappPacket) GetLengthBytes() []byte {
	return packet.header[lengthBytesIndex : lengthBytesIndex+lengthByteLength]
}

//GetCheckSum returns the checksum byte in the header
func (packet *DappPacket) GetCheckSum() byte { return packet.header[checkSumByteIndex] }

//GetHeaderCheckSum returns the header checksum byte in the header
func (packet *DappPacket) GetHeaderCheckSum() byte { return packet.header[headerCheckSumByteIndex] }

//GetPacketDataLength returns the length of the data section in bytes.
func (packet *DappPacket) GetPacketDataLength() int {
	l := *new(big.Int).SetBytes(packet.GetLengthBytes())
	return int(l.Uint64())
}

//GetLength returns the total length of the packet in bytes
func (packet *DappPacket) GetLength() int {
	return len(packet.header) + len(packet.data)
}

//IsBroadcast returns if the packet is a broadcast
func (packet *DappPacket) IsBroadcast() bool {
	return packet.GetBroadcastByte() == broadcastByte
}

//GetRawBytes returns the whole packet in raw bytes
func (packet *DappPacket) GetRawBytes() []byte {
	return append(packet.header, packet.data...)
}

//verifyHeader verifies if the header bytes are correct
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

//verifyDataChecksum verifies if the data checksum is correct
func (packet *DappPacket) verifyDataChecksum() error {
	dataCheckSum := checkSum(packet.data)

	if dataCheckSum != packet.GetCheckSum() {
		return ErrCheckSumIncorrect
	}

	return nil
}

//containStartingBytes checks if the DappPacket contains starting bytes
func (packet *DappPacket) containStartingBytes() bool {
	if len(packet.header) < startByteLength {
		return false
	}

	return reflect.DeepEqual(packet.GetStartBytes(), startBytes)
}

//constructHeader constructs the header bytes from the given data bytes
func constructHeader(data []byte, isBroadcast bool) []byte {

	length := len(data)
	msg := make([]byte, lengthByteLength)
	lengthBytes := big.NewInt(int64(length)).Bytes()
	lenDiff := len(msg) - len(lengthBytes)
	for i, b := range lengthBytes {
		msg[i+lenDiff] = b
	}

	header := append(startBytes, msg...)

	isBroadcastByte := unitcastByte
	if isBroadcast {
		isBroadcastByte = broadcastByte
	}

	header = append(header, isBroadcastByte)

	cs := checkSum(data)
	header = append(header, cs)
	headerCs := checkSum(header)
	header = append(header, headerCs)
	return header
}

//checkSum returns the checksum of a slice of bytes
func checkSum(data []byte) byte {
	sum := byte(0)
	for _, d := range data {
		sum += d
	}
	return sum
}
