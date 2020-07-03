package secp256r1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// ECDSASign ECDSA数字签名[]byte

//func Sign(hashed []byte, privateKey ecdsa.PrivateKey) ([]byte,error) {
func Sign(hashed []byte, priKey []byte) ([]byte,error) {
	privateKey ,err := ToECDSAPrivateKey(priKey)
	if err != nil {
		return nil,nil
	}
	// 1、数字签名生成r、s的big.Int对象，参数是随机数、私钥、签名文件的哈希串
	//privateKey.PublicKey.Curve = elliptic.P256()
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashed)
	if err != nil {
		return nil,nil
	}
	sign :=append(r.Bytes(),s.Bytes()...)
	fmt.Println(len(r.Bytes()))
	fmt.Println(len(s.Bytes()))

	pub := append(privateKey.X.Bytes(),privateKey.Y.Bytes()...)
	//flag ,_ := VerifySign(hashed,sign,&privateKey.PublicKey)
	flag ,_ := VerifySign(hashed,sign,pub)
	flag1 := ecdsa.Verify(&privateKey.PublicKey, hashed, r, s)
	fmt.Println("签名验证结果：", flag)
	fmt.Println("签名验证结果1：", flag1)
	return sign,nil
}

func Verify(text []byte, signatureByte []byte, publicKeyBytes []byte) (bool, error) {
	// 公钥长度
	fmt.Println("publicKeyBytes len:",len(publicKeyBytes))
	//keyLen := len(publicKeyBytes)
	//if keyLen != 65 {
	//	return false,nil
	//}
	// 1、生成椭圆曲线对象
	//curve := secp256k1.S256()
	//curve := elliptic.P256()
	//x := new(big.Int).SetBytes(publicKeyBytes[:32])
	//y := new(big.Int).SetBytes(publicKeyBytes[32:])
	//publicKey := ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	publicKey,err:= ToECDSAPublicKey(publicKeyBytes)
	if err != nil {
		return false,nil
	}
	r := new(big.Int).SetBytes(signatureByte[:32])
	s := new(big.Int).SetBytes(signatureByte[32:64])
	return ecdsa.Verify(publicKey, text, r, s),nil
}
// ToECDSAPublicKey creates a public key with the given data value.
func ToECDSAPublicKey(pub []byte) (*ecdsa.PublicKey, error) {
	if len(pub) == 0 {
		return nil, errors.New("ecdsa: please input public key bytes")
	}
	//x, y := elliptic.Unmarshal(secp256k1.S256(), pub)
	x, y := elliptic.Unmarshal(elliptic.P256(), pub)
	fmt.Println("x:",x.Bytes())
	fmt.Println("y:",y.Bytes())
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil
}

//func VerifySign(text []byte, signatureByte []byte,publicKey *ecdsa.PublicKey) (bool, error) {
func VerifySign(text []byte, signatureByte []byte, publicKeyBytes []byte) (bool, error) {
	//curve := secp256k1.S256()
	curve := elliptic.P256()
	x := new(big.Int).SetBytes(publicKeyBytes[:32])
	y := new(big.Int).SetBytes(publicKeyBytes[32:])
	//生成公钥对象
	publicKey := ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	r := new(big.Int).SetBytes(signatureByte[:32])
	s := new(big.Int).SetBytes(signatureByte[32:64])
	return ecdsa.Verify(&publicKey, text, r, s),nil
}


func ToECDSAPrivateKey(d []byte) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	//priv.PublicKey.Curve = S256() // ********************
	priv.PublicKey.Curve = elliptic.P256() // ********************
	priv.D = new(big.Int).SetBytes(d)
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	return priv, nil
}



















//func Verify(text []byte, signatureByte []byte, publicKeyBytes []byte) (bool, error) {
//	// 公钥长度
//	//keyLen := len(publicKeyBytes)
//	//if keyLen != 65 {
//	//	return false,nil
//	//}
//	// 1、生成椭圆曲线对象
//	//curve := elliptic.P256()
//	curve := secp256k1.S256()
//	// 2、根据公钥字节数字，获取公钥中的x和y
//	// 公钥字节中的前一半为x轴坐标，再将字节数组转成big.Int类型
//	//publicKeyBytes = publicKeyBytes[1:]
//	// x := big.NewInt(0).SetBytes(publicKeyBytes[:32])
//	x := new(big.Int).SetBytes(publicKeyBytes[1:33])
//	y := new(big.Int).SetBytes(publicKeyBytes[33:])
//	// 3、生成公钥对象
//	publicKey := ecdsa.PublicKey{Curve: curve, X: x, Y: y}
//	// 4、对der格式的签名进行解析，获取r/s字节数组后转成big.Int类型
//	r := new(big.Int).SetBytes(signatureByte[:32])
//	s := new(big.Int).SetBytes(signatureByte[32:64])
//	return ecdsa.Verify(&publicKey, text, r, s),nil
//}
