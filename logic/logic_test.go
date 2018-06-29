package logic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var wallet1 string
var wallet2 string

func TestCreateBlockchain(t *testing.T) {

}

func TestGetBalance(t *testing.T) {

}

func TestGetAllAddresses(t *testing.T) {

}

func TestCreateWallet(t *testing.T) {

}

func TestSend(t *testing.T) {
	err := Send(wallet1, wallet2, 10)
	assert.Nil(t, err)
}
