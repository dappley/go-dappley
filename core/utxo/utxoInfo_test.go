package utxo

import (
	utxopb "github.com/dappley/go-dappley/core/utxo/pb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewUTXOInfo(t *testing.T) {
	assert.Equal(t, &UTXOInfo{[]byte{}, []byte{}, 0}, NewUTXOInfo())
}

func TestUTXOInfo_ToProto(t *testing.T) {
	utxoInfo := &UTXOInfo{
		lastUTXOKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		createContractUTXOKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	expected := &utxopb.UtxoInfo{
		LastUtxoKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		UtxoCreateContractKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	assert.Equal(t, expected, utxoInfo.ToProto())
}

func TestUTXOInfo_FromProto(t *testing.T) {
	utxoInfoProto := &utxopb.UtxoInfo{
		LastUtxoKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		UtxoCreateContractKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	expected := &UTXOInfo{
		lastUTXOKey:           []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x30},
		createContractUTXOKey: []byte{0x74, 0x65, 0x73, 0x74, 0x5f, 0x31},
	}
	utxoInfo := &UTXOInfo{}
	utxoInfo.FromProto(utxoInfoProto)
	assert.Equal(t, expected, utxoInfo)
}
