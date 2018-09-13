package client

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core"
)

func TestWallet_ContainAddress(t *testing.T) {
	wallet := NewWallet()
	tests := []struct{
		name 	 string
		input    core.Address
		expected bool
	}{{"contains address",wallet.GetAddress(), true},
	  {"does not contain address",core.Address{},false},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){

			assert.Equal(t,tt.expected, wallet.ContainAddress(tt.input))
		})
	}
}
