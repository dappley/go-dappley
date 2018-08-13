package consensus

import (
	"testing"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
	"math/big"
)

func TestMiner_VerifyNonce(t *testing.T){

	miner := NewMiner()
	cbAddr := core.Address{"121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"}
	bc := core.CreateBlockchain(
		cbAddr,
		storage.NewRamStorage(),
		nil,
	)
	defer bc.DB.Close()

	miner.Setup(bc,cbAddr.Address, nil)

	//prepare a block with correct nonce value
	newBlock := core.NewBlock(nil,nil)
	nonce := int64(0)
	mineloop2:
	for{
		if hash, ok := miner.verifyNonce(nonce, newBlock); ok {
			newBlock.SetHash(hash)
			newBlock.SetNonce(nonce)
			break mineloop2
		}else{
			nonce++
		}
	}

	//check if the verifyNonce function returns true
	_, ok := miner.verifyNonce(nonce, newBlock)
	assert.True(t, ok)

	//input a wrong nonce value, check if it returns false
	_, ok = miner.verifyNonce(nonce-1, newBlock)
	assert.False(t, ok)
}

func TestMiner_SetTargetBit(t *testing.T) {
	tests := []struct{
		name 	 string
		bit 	 int
		expected int
	}{{"regular",16,16},
		{"zero",0,16},
		{"negative",-5,16},
		{"above256",257,16},
		{"regular2",18,18},
		{"equalTo256",256,256},
	}

	miner := NewMiner()
	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){
			miner.SetTargetBit(tt.bit)
			target := big.NewInt(1)
			target.Lsh(target,uint(256-tt.expected))
			assert.Equal(t,target,miner.target)
		})
	}
}