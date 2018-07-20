package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
	"github.com/dappley/go-dappley/storage"
)

func TestBlockPool_VerifyHeight(t *testing.T) {
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()

	//blk2 should not pass the height verification since it has the same height as blk1
	assert.False(t, verifyHeight(blk1, blk2))

	//set the height of second block to be 1 higher than the first block
	blk2.height = blk1.height + 1
	//then blk2 should pass the height verification
	assert.True(t, verifyHeight(blk1, blk2))
}

func TestBlockPool_VerifyLastBlockHash(t *testing.T) {
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()

	//blk2 should not pass the lastblock hash verification since it has the same height as blk1
	assert.False(t, verifyLastBlockHash(blk1, blk2))

	//set the prevHash of third block to be the hash value of the first blk
	blk3 := &Block{
		header: &BlockHeader{
			hash: 		[]byte("Hash3"),
			prevHash: 	blk1.GetHash(),
			nonce:		0,
			timestamp:  time.Now().Unix(),
		},
		transactions: nil,
	}

	//then blk3 should pass the lastblock hash verification
	assert.True(t, verifyLastBlockHash(blk1, blk3))
}

func TestBlockPool_VerifyBlock(t *testing.T) {
	blk1 := GenerateMockBlock()
	blk2 := GenerateMockBlock()
	blk3 := &Block{
		header: &BlockHeader{
			hash: 		[]byte("Hash3"),
			prevHash: 	blk1.GetHash(),
			nonce:		0,
			timestamp:  time.Now().Unix(),
		},
		transactions: nil,
		height:		  0,
	}

	//blk2 should not pass the verification.
	//its height is not 1 more than blk1; its previous hash it not the hash value for blk1
	assert.False(t,verifyBlock(blk1,blk2))

	//set the height of second block to be 1 higher than the first block
	blk2.height = blk1.height + 1
	//blk2 still should not pass verification due to previous hash value
	assert.False(t, verifyBlock(blk1, blk2))

	//blk3 should not pass the verification since its height is not 1 more than blk1
	assert.False(t,verifyBlock(blk1,blk3))

	//set the height of second block to be 1 higher than the first block
	blk3.height = blk1.height + 1

	//blk3 should not pass the verification since its hash is not correct
	assert.False(t, verifyBlock(blk1,blk3))

	//calculate correct hash value for blk3
	hash := blk3.CalculateHash()
	blk3.SetHash(hash)

	//now blk3 should pass the verification
	assert.True(t, verifyBlock(blk1,blk3))
}

func TestBlockPool_GetBlockchain(t *testing.T) {
	db := storage.NewRamStorage()
	defer db.Close()
	addr := Address{"17DgRtQVvaytkiKAfXx9XbV23MESASSwUz"}
	bc, err:= CreateBlockchain(addr,db)
	assert.Nil(t, err)
	hash1, err:= bc.GetLastHash()
	assert.Nil(t, err)
	newbc := bc.blockPool.GetBlockchain()

	hash2, err := newbc.GetLastHash()
	assert.Nil(t, err)
	assert.ElementsMatch(t,hash1, hash2)

}