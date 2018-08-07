package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeerlist_ToProto(t *testing.T) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}

	pl := NewPeerListStr(strs)
	pl1 := NewPeerList(nil)
	pl1.FromProto(pl.ToProto())
	assert.ElementsMatch(t, pl.GetPeerlist(), pl1.GetPeerlist())
}

func TestNewPeerlistStr(t *testing.T) {
	//two duplicated addresses
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList
	pl := NewPeerListStr(strs)

	//the duplicated address should be filtered out
	assert.Equal(t, 2, len(pl.GetPeerlist()))
}

func TestPeerlist_IsInPeerlist(t *testing.T) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUGBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl := NewPeerListStr(strs)
	ps := []*Peer{}
	for _, s := range strs {
		p, err := CreatePeerFromString(s)
		assert.Nil(t, err)
		ps = append(ps, p)
		//any of the 3 addresses above should be contained in the list
		assert.True(t, pl.IsInPeerlist(p))
	}

	//create a new multiaddress
	newStr := "/ip4/192.168.10.106/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	p, err := CreatePeerFromString(newStr)
	assert.Nil(t, err)
	//it should not be in the list
	assert.False(t, pl.IsInPeerlist(p))
}

func TestPeerlist_AddNonDuplicate(t *testing.T) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUTBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl := NewPeerListStr(strs)
	newStr := "/ip4/192.168.10.106/tcp/10000/ipfs/QmWvMUaBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"

	ps := []*Peer{}
	for _, s := range strs {
		p, err := CreatePeerFromString(s)
		assert.Nil(t, err)
		ps = append(ps, p)
		//any of the 3 addresses above should be contained in the list
		assert.True(t, pl.IsInPeerlist(p))
	}
	//add the fourth address
	p, err := CreatePeerFromString(newStr)
	assert.Nil(t, err)
	ps = append(ps, p)
	pl.Add(p)

	//the final peerList should contain all 4 addresses
	assert.ElementsMatch(t, ps, pl.GetPeerlist())
}

func TestPeerlist_AddDuplicate(t *testing.T) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUABeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl := NewPeerListStr(strs)
	newStr := "/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	strs = append(strs, newStr)
	ps := []*Peer{}
	for _, s := range strs {
		p, err := CreatePeerFromString(s)
		assert.Nil(t, err)
		ps = append(ps, p)
		//any of the 3 addresses above should be contained in the list
		assert.True(t, pl.IsInPeerlist(p))
	}
	//add the fourth address
	pl.Add(ps[3])

	//the final peerList should contain all 4 addresses
	assert.ElementsMatch(t, ps[:3], pl.GetPeerlist())
}

func TestPeerlist_MergePeerlist(t *testing.T) {
	strs1 := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvsUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl1 := NewPeerListStr(strs1)

	strs2 := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWvrUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl2 := NewPeerListStr(strs2)

	pl1.MergePeerlist(pl2)

	//expected result. The repeated address should be filtered out
	expectedStrs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvsUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWvrUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}

	expectedPl := NewPeerListStr(expectedStrs)

	assert.ElementsMatch(t, expectedPl.GetPeerlist(), pl1.GetPeerlist())

}

func TestPeerlist_FindNewPeers(t *testing.T) {
	strs1 := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWeMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl1 := NewPeerListStr(strs1)

	strs2 := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWjMUtBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWqMUqBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl2 := NewPeerListStr(strs2)

	retpl := pl1.FindNewPeers(pl2)

	//expected result. The repeated address should be filtered out
	expectedStrs := []string{
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWjMUtBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10001/ipfs/QmWqMUqBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	expectedPl := NewPeerListStr(expectedStrs)

	assert.ElementsMatch(t, expectedPl.GetPeerlist(), retpl.GetPeerlist())
}

func TestPeerlist_AddMoreThanLimit(t *testing.T) {
	strs1 := []string{
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.101/tcp/10000/ipfs/QmWeMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.102/tcp/10000/ipfs/QmWaMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.103/tcp/10000/ipfs/QmWdMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.104/tcp/10000/ipfs/QmWcMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWqMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWqAUMBeWxwU4R3ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.107/tcp/10000/ipfs/QmWsMUMBeWxwU6R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.108/tcp/10000/ipfs/QmdhMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.111/tcp/10000/ipfs/QmakMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.112/tcp/10000/ipfs/QmWwMUMBeWxwU4R3ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.113/tcp/10000/ipfs/QmWzMUMBeWxwU4R6ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.114/tcp/10000/ipfs/QmWmMUMBeWxwU4R5ukBiKmSiGT5cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.115/tcp/10000/ipfs/QmWwMZMBeWxwU4R7ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.116/tcp/10000/ipfs/QmWwadMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.117/tcp/10000/ipfs/QmWwMUNBeWxwU4R6ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.118/tcp/10000/ipfs/QmWwMUMAeWxwU3R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.119/tcp/10000/ipfs/QmWwKUMBeWxwU4RrukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.120/tcp/10000/ipfs/QmWwMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}

	pl1 := NewPeerListStr(strs1)

	assert.Equal(t, 19, len(pl1.peers))
	strs2 := []string{
		"/ip4/192.168.10.131/tcp/10000/ipfs/QmWeMUMZeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.122/tcp/10000/ipfs/QmWeMUMKeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.123/tcp/10000/ipfs/QmWeMUMQeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	pl2 := NewPeerListStr(strs2)

	newStr1 := "/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	newStr2 := "/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUkBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	p1, _ := CreatePeerFromString(newStr1)

	pl1.Add(p1)

	assert.Equal(t, 20, len(pl1.peers))

	p2, _ := CreatePeerFromString(newStr2)

	pl1.Add(p2)

	assert.Equal(t, 20, len(pl1.peers))

	pl2.AddMultiple(pl1.peers)
	assert.Equal(t, 20, len(pl2.peers))
}

func TestPeerList_IsFull(t *testing.T) {
	strs1 := []string{
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.101/tcp/10000/ipfs/QmWeMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.102/tcp/10000/ipfs/QmWaMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.103/tcp/10000/ipfs/QmWdMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.104/tcp/10000/ipfs/QmWcMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWqMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWqAUMBeWxwU4R3ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.107/tcp/10000/ipfs/QmWsMUMBeWxwU6R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.108/tcp/10000/ipfs/QmdhMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.111/tcp/10000/ipfs/QmakMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.112/tcp/10000/ipfs/QmWwMUMBeWxwU4R3ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.113/tcp/10000/ipfs/QmWzMUMBeWxwU4R6ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.114/tcp/10000/ipfs/QmWmMUMBeWxwU4R5ukBiKmSiGT5cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.115/tcp/10000/ipfs/QmWwMZMBeWxwU4R7ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.116/tcp/10000/ipfs/QmWwadMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.117/tcp/10000/ipfs/QmWwMUNBeWxwU4R6ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.118/tcp/10000/ipfs/QmWwMUMAeWxwU3R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.119/tcp/10000/ipfs/QmWwKUMBeWxwU4RrukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.120/tcp/10000/ipfs/QmWwMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	pl1 := NewPeerListStr(strs1)
	assert.False(t, pl1.ListIsFull())

	strs2 := []string{
		"/ip4/192.168.10.131/tcp/10000/ipfs/QmWeMUMZeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.122/tcp/10000/ipfs/QmWeMUMKeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.123/tcp/10000/ipfs/QmWeMUMQeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	pl2 := NewPeerListStr(strs2)
	pl1.AddMultiple(pl2.peers)
	assert.True(t, pl1.ListIsFull())
}

func TestPeerList_RemoveOneIP(t *testing.T) {
	strs1 := []string{
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWyMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.101/tcp/10000/ipfs/QmWeMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.102/tcp/10000/ipfs/QmWaMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.103/tcp/10000/ipfs/QmWdMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.104/tcp/10000/ipfs/QmWcMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWqMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.106/tcp/10000/ipfs/QmWqAUMBeWxwU4R3ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.107/tcp/10000/ipfs/QmWsMUMBeWxwU6R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.108/tcp/10000/ipfs/QmdhMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.111/tcp/10000/ipfs/QmakMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.112/tcp/10000/ipfs/QmWwMUMBeWxwU4R3ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.113/tcp/10000/ipfs/QmWzMUMBeWxwU4R6ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.114/tcp/10000/ipfs/QmWmMUMBeWxwU4R5ukBiKmSiGT5cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.115/tcp/10000/ipfs/QmWwMZMBeWxwU4R7ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.116/tcp/10000/ipfs/QmWwadMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.117/tcp/10000/ipfs/QmWwMUNBeWxwU4R6ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.118/tcp/10000/ipfs/QmWwMUMAeWxwU3R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.119/tcp/10000/ipfs/QmWwKUMBeWxwU4RrukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.120/tcp/10000/ipfs/QmWwMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}

	pl1 := NewPeerListStr(strs1)

	assert.Equal(t, 19, len(pl1.peers))
	newStr1 := "/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	newStr2 := "/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUkBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	p1, _ := CreatePeerFromString(newStr1)

	pl1.Add(p1)

	assert.Equal(t, 20, len(pl1.peers))

	p2, _ := CreatePeerFromString(newStr2)

	pl1.Add(p2)

	assert.Equal(t, 20, len(pl1.peers))
	pl1.RemoveOneIP(p2)
	assert.Equal(t, 19, len(pl1.peers))
	assert.False(t, pl1.IsInPeerlist(p2))
	pl1.Add(p2)
	assert.Equal(t, 20, len(pl1.peers))
	assert.True(t, pl1.IsInPeerlist(p2))
}
