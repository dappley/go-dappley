package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multiaddr"
)

type retFormat struct{
	peerid  string
	addr	string
}

func TestPeer_ToProto(t *testing.T){
	peerid, _ := peer.IDB58Decode("QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ")
	addr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")
	p := &Peer{peerid,addr}
	pb := &networkpb.Peer{
		Peerid: "QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		Addr:	"/ip4/127.0.0.1/tcp/10000",
	}
	assert.Equal(t,pb,p.ToProto())
}

func TestPeer_FromProto(t *testing.T){
	peerid, _ := peer.IDB58Decode("QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ")
	addr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")
	p1 := &Peer{peerid,addr}
	pb := &networkpb.Peer{
		Peerid: "QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		Addr:	"/ip4/127.0.0.1/tcp/10000",
	}
	p2 := &Peer{}
	p2.FromProto(pb)
	assert.Equal(t,p1,p2)
}

func TestPeerlist_ToProto(t *testing.T) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvFUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvGUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	pl := NewPeerListStr(strs)

	plpb := &networkpb.Peerlist{
		Peerlist:  []*networkpb.Peer{
			{
				Peerid: "QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				Addr:   "/ip4/127.0.0.1/tcp/10000",
			},
			{
				Peerid: "QmWvFUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				Addr:   "/ip4/192.168.10.110/tcp/10000",
			},
			{
				Peerid: "QmWvGUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				Addr:   "/ip4/192.168.10.105/tcp/10000",
			},
		},
	}
	assert.Equal(t, plpb, pl.ToProto())
}

func TestPeerlist_FromProto(t *testing.T) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvFUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvGUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	pl := NewPeerListStr(strs)

	plpb := &networkpb.Peerlist{
		Peerlist:  []*networkpb.Peer{
			{
				Peerid: "QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				Addr:   "/ip4/127.0.0.1/tcp/10000",
			},
			{
				Peerid: "QmWvFUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				Addr:   "/ip4/192.168.10.110/tcp/10000",
			},
			{
				Peerid: "QmWvGUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				Addr:   "/ip4/192.168.10.105/tcp/10000",
			},
		},
	}
	pl1 := &PeerList{}
	pl1.FromProto(plpb)
	assert.Equal(t, pl, pl1)
}

func TestNewPeerlistStr(t *testing.T) {

	//create a test struct that contains all possible inputs and its expected output
	tests:=[]struct{
		name			string
		addrs			[]string
		expectedAddr	[]retFormat
	}{
		{
			name:			"normal_input",
			addrs:			[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			expectedAddr:	[]retFormat{
				{
					peerid: "<peer.ID WvMUNB>",
					addr:	"/ip4/127.0.0.1/tcp/10000",
				},
				{
					peerid: "<peer.ID WvMUMB>",
					addr:	"/ip4/192.168.10.110/tcp/10000",
				},
			},
		},
		{
			name:			"duplicated_input",
			addrs:			[]string{
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			expectedAddr:	[]retFormat{
				{
					peerid: "<peer.ID WvMUMB>",
					addr:	"/ip4/192.168.10.110/tcp/10000",
				},
			},
		},
		{
			name:			"invalid_input",
			addrs:			[]string{
				"T8cDqmkfrXCb2qTVHpofJ",
			},
			expectedAddr:	[]retFormat{
			},
		},
		{
			name:			"partially_invalid_input",
			addrs:			[]string{
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"T8cDqmkfrXCb2qTVHpofJ",
			},
			expectedAddr:	[]retFormat{
				{
					peerid: "<peer.ID WvMUMB>",
					addr:	"/ip4/192.168.10.110/tcp/10000",
				},
			},
		},
		{
			name:			"no_input",
			addrs:			[]string{
			},
			expectedAddr:	[]retFormat{
			},
		},
	}

	//run tests
	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){
			pl:=NewPeerListStr(tt.addrs)
			//if the expectedAddr is empty, it means the peerlist is expected to be empty
			if len(tt.expectedAddr) == 0 {
				assert.Empty(t,pl.peers)
			}else{
				for i,peer := range pl.GetPeerlist(){
					assert.Equal(t,tt.expectedAddr[i].peerid,peer.peerid.String())
					assert.Equal(t,tt.expectedAddr[i].addr, peer.addr.String())
				}
			}
		})
	}

}

func TestPeerlist_IsInPeerlist(t *testing.T) {
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
		"/ip4/192.168.10.105/tcp/10000/ipfs/QmWvMUGBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	//create new peerList with 3 addrs
	pl := NewPeerListStr(strs)

	tests := []struct{
		name 		string
		pid 		string
		addr		string
		expected	bool
	}{
		{
			name:		"InPeerList",
			pid:		"QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr: 		"/ip4/127.0.0.1/tcp/10000",
			expected:	true,
		},
		{
			name:		"NotInPeerList",
			pid:		"QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr:		"/ip4/192.168.10.106/tcp/10000",
			expected:	false,
		},
		{
			name:		"OnlyPidInPeerList",
			pid:		"QmWvMUNBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr:		"/ip4/192.168.10.106/tcp/10000",
			expected:	true,
		},
		{
			name:		"OnlyAddrNotInPeerList",
			pid:		"QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr:		"/ip4/127.0.0.1/tcp/10000",
			expected:	true,
		},
		{
			name: 		"NoInput",
			pid:		"",
			addr: 		"",
			expected:	false,
		},
		{
			name: 		"InvalidInput",
			pid:		"dfdf",
			addr:		"dfdf",
			expected:	false,
		},
	}

	for _,tt := range tests{
		t.Run(tt.name, func(t *testing.T){
			peerid, _ := peer.IDB58Decode(tt.pid)
			addr, _ := multiaddr.NewMultiaddr(tt.addr)
			p := &Peer{
				peerid: peerid,
				addr:	addr,
			}
			assert.Equal(t,tt.expected,pl.IsInPeerlist(p))
		})
	}
}

func TestPeerList_Add(t *testing.T){
	//create a peer list
	strs := []string{
		"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
	}
	pl := NewPeerListStr(strs)

	tests := []struct{
		name 		string
		pid 		string
		addr 		string
		expected	retFormat
	}{
		{
			name:		"normal",
			pid:		"QmWvMUSBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr:		"/ip4/192.168.10.110/tcp/10000",
			expected:	retFormat{
				peerid: "<peer.ID WvMUSB>",
				addr:	"/ip4/192.168.10.110/tcp/10000",
			},
		},
		{
			name:		"duplicated",
			pid:		"QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr:		"/ip4/127.0.0.1/tcp/10000",
			expected:	retFormat{
				peerid: "<peer.ID WvMUSB>",
				addr:	"/ip4/192.168.10.110/tcp/10000",
			},
		},
		{
			name:		"duplicated_pid",
			pid:		"QmWvMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr:		"/ip4/127.0.0.2/tcp/10000",
			expected:	retFormat{
				peerid: "<peer.ID WvMUSB>",
				addr:	"/ip4/192.168.10.110/tcp/10000",
			},
		},
		{
			name:		"duplicated_addr",
			pid:		"QmWsMUDBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			addr:		"/ip4/127.0.0.1/tcp/10000",
			expected:	retFormat{
				peerid: "<peer.ID WvMUSB>",
				addr:	"/ip4/192.168.10.110/tcp/10000",
			},
		},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){
			peerid, _ := peer.IDB58Decode(tt.pid)
			addr, _ := multiaddr.NewMultiaddr(tt.addr)
			p := &Peer{peerid,addr}
			pl.Add(p)
			assert.Equal(t, tt.expected.peerid, pl.peers[len(pl.peers)-1].peerid.String())
			assert.Equal(t, tt.expected.addr, pl.peers[len(pl.peers)-1].addr.String())
		})
	}

}

func TestPeerlist_MergePeerlist(t *testing.T) {

	tests := []struct{
		name 		string
		peerStr1	[]string
		peerStr2	[]string
		expStr 		[]string
	}{
		{
			name:		"Normal",
			peerStr1: 	[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			peerStr2:	[]string{
				"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			expStr:		[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
		},
		{
			name:		"Overlapping",
			peerStr1: 	[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			peerStr2:	[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			expStr:		[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.106/tcp/10000/ipfs/QmWgMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
		},
		{
			name:		"Duplicated",
			peerStr1: 	[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			peerStr2:	[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			expStr:		[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
		},
		{
			name:		"NoInput",
			peerStr1: 	[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
			peerStr2:	[]string{
			},
			expStr:		[]string{
				"/ip4/127.0.0.1/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
				"/ip4/192.168.10.110/tcp/10000/ipfs/QmWvaUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ",
			},
		},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){
			pl1 := NewPeerListStr(tt.peerStr1)
			pl2 := NewPeerListStr(tt.peerStr2)
			pl1.MergePeerlist(pl2)
			expectedPl := NewPeerListStr(tt.expStr)
			assert.ElementsMatch(t, expectedPl.GetPeerlist(), pl1.GetPeerlist())
		})
	}
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

	newStr1 := "/ip4/192.168.10.121/tcp/10000/ipfs/QmNvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
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
	newStr1 := "/ip4/192.168.10.121/tcp/10000/ipfs/QmWvMUMBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
	newStr2 := "/ip4/192.168.10.122/tcp/10000/ipfs/QmWvMUkBeWxwU4R5ukBiKmSiGT8cDqmkfrXCb2qTVHpofJ"
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
