package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"math/big"
	mrand "math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/dappley/go-dappley/crypto/hash"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/stretchr/testify/assert"
	//"fmt"
	//"crypto/sha256"
)

func TestMain(m *testing.M) {
	// run tests
	code := m.Run()
	os.Exit(code)
}

func TestSha3256(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name       string
		args       args
		wantDigest []byte
	}{
		{
			"blank string",
			args{[]byte("")},
			[]byte{167, 255, 198, 248, 191, 30, 215, 102, 81, 193, 71, 86, 160, 97, 214, 98, 245, 128, 255, 77, 228, 59, 73, 250, 130, 216, 10, 75, 128, 248, 67, 74},
		},
		{
			"Hello, world",
			args{[]byte("Hello, world")},
			[]byte{53, 80, 171, 169, 116, 146, 222, 56, 175, 48, 102, 240, 21, 127, 197, 50, 219, 103, 145, 179, 125, 83, 38, 44, 231, 104, 141, 204, 93, 70, 24, 86},
		},
		{
			"hello达扑",
			args{[]byte("hello达扑")},
			[]byte{65, 80, 171, 183, 137, 18, 127, 143, 23, 5, 97, 178, 217, 188, 23, 166, 201, 238, 195, 110, 203, 122, 174, 108, 29, 130, 2, 0, 220, 67, 0, 114},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotDigest := hash.Sha3256(tt.args.bytes); !reflect.DeepEqual(gotDigest, tt.wantDigest) {
				t.Errorf("Sha3256() = %v, want %v", gotDigest, tt.wantDigest)
			}
		})
	}
	//more than one input
	t.Run("more than one byte arrays", func(t *testing.T) {
		param1 := []byte("qqqqq")
		param2 := []byte("wwwww")
		want_res := []byte{0, 74, 152, 217, 236, 38, 51, 33, 201, 140, 20, 186, 209, 196, 178, 240, 239, 194, 241, 16, 29, 44, 247, 139, 57, 54, 81, 142, 135, 170, 146, 144}
		act_res := hash.Sha3256(param1, param2)
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Sha3256() = %v, want %v", act_res, want_res)
		}
	})
	//input nil
	t.Run("input nil", func(t *testing.T) {
		act_res := hash.Sha3256(nil)
		want_res := []byte{167, 255, 198, 248, 191, 30, 215, 102, 81, 193, 71, 86, 160, 97, 214, 98, 245, 128, 255, 77, 228, 59, 73, 250, 130, 216, 10, 75, 128, 248, 67, 74}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Sha3256() = %v, want %v", act_res, want_res)
		}
	})
	//input []byte
	t.Run("input []byte", func(t *testing.T) {
		var tn []byte
		act_res := hash.Sha3256(tn)
		want_res := []byte{167, 255, 198, 248, 191, 30, 215, 102, 81, 193, 71, 86, 160, 97, 214, 98, 245, 128, 255, 77, 228, 59, 73, 250, 130, 216, 10, 75, 128, 248, 67, 74}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Sha3256() = %v, want %v", act_res, want_res)
		}
	})
}

func TestRipemd160(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name       string
		args       args
		wantDigest []byte
	}{
		{
			"blank string",
			args{[]byte("")},
			[]byte{156, 17, 133, 165, 197, 233, 252, 84, 97, 40, 8, 151, 126, 232, 245, 72, 178, 37, 141, 49},
		},
		{
			"The quick brown fox jumps over the lazy dog",
			args{[]byte("The quick brown fox jumps over the lazy dog")},
			[]byte{55, 243, 50, 246, 141, 183, 123, 217, 215, 237, 212, 150, 149, 113, 173, 103, 28, 249, 221, 59},
		},
		{
			"The quick brown fox jumps over the lazy cog",
			args{[]byte("The quick brown fox jumps over the lazy cog")},
			[]byte{19, 32, 114, 223, 105, 9, 51, 131, 94, 184, 182, 173, 11, 119, 231, 182, 241, 74, 202, 215},
		},
		{
			"hello达扑",
			args{[]byte("hello达扑")},
			[]byte{167, 195, 35, 3, 84, 139, 126, 26, 168, 131, 100, 229, 19, 96, 242, 53, 148, 61, 123, 134},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotDigest := hash.Ripemd160(tt.args.bytes); !reflect.DeepEqual(gotDigest, tt.wantDigest) {
				t.Errorf("Ripemd160() = %v, want %v", gotDigest, tt.wantDigest)
			}
		})
	}
	//more than one input
	t.Run("more than one byte arrays", func(t *testing.T) {
		param1 := []byte("qqqqq")
		param2 := []byte("hhhhhhh")
		want_res := []byte{85, 119, 150, 213, 198, 133, 95, 216, 1, 148, 125, 17, 149, 233, 108, 235, 153, 162, 80, 175}
		act_res := hash.Ripemd160(param1, param2)
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Ripemd160() = %v, want %v", act_res, want_res)
		}
	})
	//input nil
	t.Run("input nil", func(t *testing.T) {
		act_res := hash.Ripemd160(nil)
		want_res := []byte{156, 17, 133, 165, 197, 233, 252, 84, 97, 40, 8, 151, 126, 232, 245, 72, 178, 37, 141, 49}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Ripemd160() = %v, want %v", act_res, want_res)
		}
	})
	//input []byte
	t.Run("input []byte", func(t *testing.T) {
		var tn []byte
		act_res := hash.Ripemd160(tn)
		want_res := []byte{156, 17, 133, 165, 197, 233, 252, 84, 97, 40, 8, 151, 126, 232, 245, 72, 178, 37, 141, 49}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Ripemd160() = %v, want %v", act_res, want_res)
		}
	})
}

func TestBase58Encode(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name       string
		args       args
		wantDigest string
	}{
		{
			"blank string",
			args{[]byte("")},
			"",
		},
		{
			"Hello, World!",
			args{[]byte("Hello, World!")},
			"72k1xXWG59fYdzSNoA",
		},
		{
			"  Vancouver Great!  ",
			args{[]byte("  Vancouver Great!  ")},
			"Sxdywfc6JJmLjLgFzi6GMw6cBpB",
		},
		{
			"hello达扑",
			args{[]byte("hello达扑")},
			"StV1DL7vfyEzf3E",
		},
		{
			"Please show me all the utxos of this transaction",
			args{[]byte("Please show me all the utxos of this transaction")},
			"3x7F1LyM5L7rQTsSeiq3RuzTKjMTbT8B2RVBdrmqVtajCeXcrH1AhaJdiy4aghZR7K",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotDigest := base58.Encode(tt.args.bytes); !reflect.DeepEqual(gotDigest, tt.wantDigest) {
				t.Errorf("Base58Encode() = %v, want %v", gotDigest, tt.wantDigest)
			}
		})
	}
	//input nil
	t.Run("input nil", func(t *testing.T) {
		act_res := base58.Encode(nil)
		want_res := ""
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Base58Encode() = %v, want %v", act_res, want_res)
		}
	})
	//input []byte
	t.Run("input []byte", func(t *testing.T) {
		var tn []byte
		act_res := base58.Encode(tn)
		want_res := ""
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Base58Encode() = %v, want %v", act_res, want_res)
		}
	})
}

func TestBase58Decode(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name       string
		args       args
		wantDigest []byte
	}{
		{
			"blank string",
			args{""},
			[]byte(""),
		},
		{
			"Hello, World!",
			args{"72k1xXWG59fYdzSNoA"},
			[]byte("Hello, World!"),
		},
		{
			"  Vancouver Great!  ",
			args{"Sxdywfc6JJmLjLgFzi6GMw6cBpB"},
			[]byte("  Vancouver Great!  "),
		},
		{
			"hello达扑",
			args{"StV1DL7vfyEzf3E"},
			[]byte("hello达扑"),
		},
		{
			"Please show me all the utxos of this transaction",
			args{"3x7F1LyM5L7rQTsSeiq3RuzTKjMTbT8B2RVBdrmqVtajCeXcrH1AhaJdiy4aghZR7K"},
			[]byte("Please show me all the utxos of this transaction"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotDigest := []byte(base58.Decode(tt.args.str)); !reflect.DeepEqual(gotDigest, tt.wantDigest) {
				t.Errorf("Base58Decode() = %v, want %v", gotDigest, tt.wantDigest)
			}
		})
	}
}

func TestBase58Decode_fail(t *testing.T) {
	//invalid input
	t.Run("invalid input add space", func(t *testing.T) {
		act_res := base58.Decode("dw qffwqfwq")
		want_res := []byte{}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Base58Decode() = %v, want %v", act_res, want_res)
		}
	})
	t.Run("invalid input add \\t", func(t *testing.T) {
		act_res := base58.Decode("dw\tqffwqfwq")
		want_res := []byte{}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Base58Decode() = %v, want %v", act_res, want_res)
		}
	})
	t.Run("invalid input add chinese character", func(t *testing.T) {
		act_res := base58.Decode("dw的qffwqfwq")
		want_res := []byte{}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Base58Decode() = %v, want %v", act_res, want_res)
		}
	})
	t.Run("invalid input add \\n", func(t *testing.T) {
		act_res := base58.Decode("dw\nqffwqfwq")
		want_res := []byte{}
		if !reflect.DeepEqual(act_res, want_res) {
			t.Errorf("Base58Decode() = %v, want %v", act_res, want_res)
		}
	})
}

func TestSecp256k1FromECDSAPriv(t *testing.T) {
	//empty privKey
	privData, err := secp256k1.FromECDSAPrivateKey(nil)
	assert.Nil(t, privData)
	assert.Equal(t, err, errors.New("ecdsa: please input private key"))

	//privKey.D bitlen far less than 256)
	t.Run("privkey with short D", func(t *testing.T) {
		var fd big.Int
		fd.SetInt64(int64(123123))
		pk, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		privKey := &ecdsa.PrivateKey{pk.PublicKey, &fd}
		privData, err = secp256k1.FromECDSAPrivateKey(privKey)
		wantDigest := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 224, 243}
		assert.Nil(t, err)
		assert.Equal(t, privData, wantDigest)
	})

	//privKey.D is equal to 0
	t.Run("privkey with D equal to 0", func(t *testing.T) {
		var fd big.Int
		fd.SetInt64(int64(0))
		pk, _ := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		privKey := &ecdsa.PrivateKey{pk.PublicKey, &fd}
		privData, err = secp256k1.FromECDSAPrivateKey(privKey)
		wantDigest := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		assert.Nil(t, err)
		assert.Equal(t, privData, wantDigest)
	})

	//fake random privkey 1
	t.Run("fake privkey 1", func(t *testing.T) {
		privKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("fakefakefakefakefakefakefakefakefakefake")))
		privData, err = secp256k1.FromECDSAPrivateKey(privKey)
		wantDigest := []byte{102, 97, 107, 101, 102, 97, 107, 101, 232, 123, 139, 153, 141, 234, 158, 246, 4, 30, 205, 160, 212, 219, 15, 32, 208, 159, 66, 2, 90, 115, 237, 38}
		assert.Nil(t, err)
		assert.Equal(t, privData, wantDigest)
	})

	//fake random privkey 2
	t.Run("fake privkey 2", func(t *testing.T) {
		privKey, _ := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader([]byte("fakemsgfakemsgfakemsgfakemsgfakemsgmmmmm")))
		privData, err = secp256k1.FromECDSAPrivateKey(privKey)
		wantDigest := []byte{97, 107, 101, 109, 115, 103, 102, 97, 237, 127, 141, 167, 151, 235, 167, 146, 44, 236, 67, 252, 97, 161, 32, 131, 225, 192, 243, 37, 19, 206, 173, 238}
		assert.Nil(t, err)
		assert.Equal(t, privData, wantDigest)
	})

}

func TestSecp256k1Sign(t *testing.T) {
	sk := secp256k1.NewSeckey()
	var msg []byte

	//random attempts
	for i := 0; i < 10; i++ {
		msg = []byte{}
		for j := 0; j < 32; j++ {
			msg = append(msg, byte(mrand.Intn(256)))
		}
		sk = secp256k1.NewSeckey()
		t.Run("random attempt "+strconv.Itoa(i), func(t *testing.T) {
			sig, err := secp256k1.Sign(msg, sk)
			assert.NotEmpty(t, sig)
			assert.Nil(t, err)
		})
	}
}

func TestSecp256k1Sign_fail(t *testing.T) {
	sk := secp256k1.NewSeckey()
	var msg []byte
	//fmt.Println("NewSeckey: ", sk)
	//msg too short
	t.Run("msg too short", func(t *testing.T) {
		msg = []byte{102, 97, 107, 101, 102, 97, 107, 101, 232, 123, 139, 153, 141, 234, 158, 246, 4, 30, 205, 160, 212, 219, 15, 32, 208, 159, 66, 2}
		sig, err := secp256k1.Sign(msg, sk)
		assert.Nil(t, sig)
		assert.Equal(t, err, secp256k1.ErrInvalidMsgLen)
	})
	//empty msg
	t.Run("empty msg", func(t *testing.T) {
		msg = []byte{}
		sig, err := secp256k1.Sign(msg, sk)
		assert.Nil(t, sig)
		assert.Equal(t, err, secp256k1.ErrInvalidMsgLen)
	})
	//nil msg
	t.Run("nil msg", func(t *testing.T) {
		msg = nil
		sig, err := secp256k1.Sign(msg, sk)
		assert.Nil(t, sig)
		assert.Equal(t, err, secp256k1.ErrInvalidMsgLen)
	})
	//msg too long
	t.Run("msg too long", func(t *testing.T) {
		msg = []byte{102, 97, 107, 101, 102, 97, 107, 101, 232, 123, 139, 153, 141, 234, 158, 246, 4, 30, 205, 160, 212, 219, 15, 32, 208, 159, 66, 2, 90, 115, 237, 38, 12, 44}
		sig, err := secp256k1.Sign(msg, sk)
		assert.Nil(t, sig)
		assert.Equal(t, err, secp256k1.ErrInvalidMsgLen)
	})
}

func TestSecp256k1Verify(t *testing.T) {

	type test struct {
		name string
		msg  []byte
		pub  []byte
		sig  []byte
	}

	tests := []test{}
	for index := 0; index < 10; index++ {
		name := "random attempt " + strconv.Itoa(index)
		msg := []byte{}
		for i := 0; i < 32; i++ {
			msg = append(msg, byte(mrand.Intn(256)))
		}
		sk := secp256k1.NewSeckey()
		pub, _ := secp256k1.GetPublicKey(sk)
		sig, _ := secp256k1.Sign(msg, sk)
		test := test{name, msg, pub, sig}
		tests = append(tests, test)
	}
	//random attempts
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := secp256k1.Verify(tt.msg, tt.sig, tt.pub)
			assert.Equal(t, result, true)
			assert.Nil(t, err)
		})
	}
}

func TestSecp256k1Verify_fail(t *testing.T) {

	fake_pk := []byte{4, 155, 5, 232, 200, 234, 120, 189, 159, 5, 74, 24, 203, 8, 216, 236, 245, 21, 14, 26, 16, 42, 223, 67, 216, 91, 151, 57, 71, 213, 1, 212, 36, 19, 40, 237, 184, 228, 34, 186, 175, 177, 102, 204, 176, 246, 159, 250, 200, 117, 254, 59, 72, 208, 141, 195, 29, 245, 149, 197, 153, 251, 131, 242, 214}
	invalid_pk := []byte{4, 155, 5, 232, 200, 234, 120, 189, 159, 5, 74, 24, 203, 8, 216, 236, 245, 21, 14, 26, 16, 42, 223, 67, 216, 91, 151, 57, 71, 213, 1, 212, 36, 19, 40, 237, 184, 228, 34, 186, 175, 177, 102, 204, 176, 246, 159, 250, 200, 117, 254, 59, 72, 208, 141, 195, 29, 245, 149}
	fake_sig := []byte{1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 28}
	invalid_sig := []byte{1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21, 19, 65, 1, 32, 231, 234, 22, 21}

	type test struct {
		name string
		msg  []byte
		pub  []byte
		sig  []byte
	}

	tests := []test{}
	for index := 0; index < 10; index++ {
		name := "random attempt " + strconv.Itoa(index)
		msg := []byte{}
		for i := 0; i < 32; i++ {
			msg = append(msg, byte(mrand.Intn(256)))
		}
		sk := secp256k1.NewSeckey()
		pub, _ := secp256k1.GetPublicKey(sk)
		sig, _ := secp256k1.Sign(msg, sk)
		test := test{name, msg, pub, sig}
		tests = append(tests, test)
	}
	//invalid public key
	for index, tt := range tests {
		tt.name = "invalid public key attempt " + strconv.Itoa(index)
		t.Run(tt.name, func(t *testing.T) {
			result, err := secp256k1.Verify(tt.msg, tt.sig, invalid_pk)
			assert.Equal(t, result, false)
			assert.Equal(t, err, secp256k1.ErrInvalidPublicKey)
		})
	}
	//wrong public key
	for index, tt := range tests {
		tt.name = "wrong public key attempt " + strconv.Itoa(index)
		t.Run(tt.name, func(t *testing.T) {
			result, err := secp256k1.Verify(tt.msg, tt.sig, fake_pk)
			assert.Equal(t, result, false)
			assert.Nil(t, err)
		})
	}
	//invalid signature length
	for index, tt := range tests {
		tt.name = "invalid signature length attempt " + strconv.Itoa(index)
		t.Run(tt.name, func(t *testing.T) {
			result, err := secp256k1.Verify(tt.msg, invalid_sig, tt.pub)
			assert.Equal(t, result, false)
			assert.Nil(t, err)
		})
	}
	//wrong signature
	for index, tt := range tests {
		tt.name = "wrong signature attempt " + strconv.Itoa(index)
		t.Run(tt.name, func(t *testing.T) {
			result, err := secp256k1.Verify(tt.msg, fake_sig, tt.pub)
			assert.Equal(t, result, false)
			assert.Nil(t, err)
		})
	}
}
