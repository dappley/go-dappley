package x509

import (
	"crypto/elliptic"
	"math/big"
	"sync"
)

// This file holds ECC curves that are not supported by the main Go crypto/elliptic
// library, but which have been observed in certificates in the wild.

var initonce sync.Once
var ecp256k1 *elliptic.CurveParams

func initAllCurves() {
	initSECP256K1()
}

func initSECP256K1() {
	// See SEC-2, section 2.2.2
	ecp256k1 = &elliptic.CurveParams{Name: "P-256-K1"}
	ecp256k1.P, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 16)
	ecp256k1.N, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	ecp256k1.B, _ = new(big.Int).SetString("0000000000000000000000000000000000000000000000000000000000000007", 16)
	ecp256k1.Gx, _ = new(big.Int).SetString("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 16)
	ecp256k1.Gy, _ = new(big.Int).SetString("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 16)
	ecp256k1.BitSize = 256
}

func secp256k1() elliptic.Curve {
	initonce.Do(initAllCurves)
	return ecp256k1
}

func unmarshal(curve elliptic.Curve, data []byte) (x, y *big.Int) {
	byteLen := (curve.Params().BitSize + 7) >> 3
	if len(data) != 1+2*byteLen {
		return
	}
	if data[0] != 4 { // uncompressed form
		return
	}
	p := curve.Params().P
	x = new(big.Int).SetBytes(data[1 : 1+byteLen])
	y = new(big.Int).SetBytes(data[1+byteLen:])
	if x.Cmp(p) >= 0 || y.Cmp(p) >= 0 {
		return nil, nil
	}

	if curve.Params().Name == "P-256-K1"{
		if !isOnCurveSecpk1(curve.Params(), x, y) {
			return nil, nil
		}
	}else {
		if !curve.IsOnCurve(x, y) {
			return nil, nil
		}
	}
	return
}

func isOnCurveSecpk1(curve *elliptic.CurveParams, x, y *big.Int) bool {
	// y² = x³ + b
	y2 := new(big.Int).Mul(y, y) //y²
	y2.Mod(y2, curve.P)          //y²%P

	x3 := new(big.Int).Mul(x, x) //x²
	x3.Mul(x3, x)                //x³

	x3.Add(x3, curve.B) //x³+B
	x3.Mod(x3, curve.P) //(x³+B)%P

	return x3.Cmp(y2) == 0
}

