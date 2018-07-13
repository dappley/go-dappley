package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestPeerlist_ToProto(t *testing.T) {
	strs:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}

	pl := NewPeerlistStr(strs)
	pl1 := NewPeerlist(nil)
	pl1.FromProto(pl.ToProto())
	assert.ElementsMatch(t, pl.GetPeerlist() , pl1.GetPeerlist())
}
