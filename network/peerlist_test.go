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

	pl := NewPeerListStr(strs)
	pl1 := NewPeerList(nil)
	pl1.FromProto(pl.ToProto())
	assert.ElementsMatch(t, pl.GetPeerlist() , pl1.GetPeerlist())
}

func TestNewPeerlistStr(t *testing.T) {
	//two duplicated addresses
	strs:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList
	pl := NewPeerListStr(strs)

	//the duplicated address should be filtered out
	assert.Equal(t,2,len(pl.GetPeerlist()))
}

func TestPeerlist_IsInPeerlist(t *testing.T) {
	strs:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUGBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl := NewPeerListStr(strs)
	ps := []*Peer{}
	for _, s := range strs{
		p, err := CreatePeerFromString(s)
		assert.Nil(t,err)
		ps = append(ps, p)
		//any of the 3 addresses above should be contained in the list
		assert.True(t,pl.IsInPeerlist(p))
	}

	//create a new multiaddress
	newStr:= "/ip4/192.168.10.106/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	p, err := CreatePeerFromString(newStr)
	assert.Nil(t,err)
	//it should not be in the list
	assert.False(t,pl.IsInPeerlist(p))
}

func TestPeerlist_AddNonDuplicate(t *testing.T) {
	strs:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUTBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl := NewPeerListStr(strs)
	newStr:= "/ip4/192.168.10.106/tcp/10000/ipfs/QmWvMUaBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"

	ps := []*Peer{}
	for _, s := range strs{
		p, err := CreatePeerFromString(s)
		assert.Nil(t,err)
		ps = append(ps, p)
		//any of the 3 addresses above should be contained in the list
		assert.True(t,pl.IsInPeerlist(p))
	}
	//add the fourth address
	p, err := CreatePeerFromString(newStr)
	assert.Nil(t,err)
	ps = append(ps, p)
	pl.Add(p)

	//the final peerList should contain all 4 addresses
	assert.ElementsMatch(t,ps,pl.GetPeerlist())
}

func TestPeerlist_AddDuplicate(t *testing.T) {
	strs:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUABeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl := NewPeerListStr(strs)
	newStr:= "/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	strs = append(strs, newStr)
	ps := []*Peer{}
	for _, s := range strs{
		p, err := CreatePeerFromString(s)
		assert.Nil(t,err)
		ps = append(ps, p)
		//any of the 3 addresses above should be contained in the list
		assert.True(t,pl.IsInPeerlist(p))
	}
	//add the fourth address
	pl.Add(ps[3])

	//the final peerList should contain all 4 addresses
	assert.ElementsMatch(t,ps[:3],pl.GetPeerlist())
}

func TestPeerlist_MergePeerlist(t *testing.T) {
	strs1:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvsUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl1 := NewPeerListStr(strs1)

	strs2:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWvrUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl2 := NewPeerListStr(strs2)

	pl1.MergePeerlist(pl2)

	//expected result. The repeated address should be filtered out
	expectedStrs:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvsUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWvrUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}

	expectedPl := NewPeerListStr(expectedStrs)

	assert.ElementsMatch(t,expectedPl.GetPeerlist(),pl1.GetPeerlist())

}

func TestPeerlist_FindNewPeers(t *testing.T) {
	strs1:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWeMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl1 := NewPeerListStr(strs1)

	strs2:= []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWjMUtBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWqMUqBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl2 := NewPeerListStr(strs2)

	retpl := pl1.FindNewPeers(pl2)

	//expected result. The repeated address should be filtered out
	expectedStrs:= []string{
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWjMUtBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWqMUqBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	expectedPl := NewPeerListStr(expectedStrs)

	assert.ElementsMatch(t,expectedPl.GetPeerlist(),retpl.GetPeerlist())
}