package encrypt

import (
	"crypto/aes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// EncryptByAes aes密码加密
func EncryptByAes(word string, keyWord string) (string, error) {
	key := []byte(keyWord)
	plaintext := []byte(word)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// ECB 模式不需要初始向量
	bs := block.BlockSize()
	plaintext = pkcs7Padding(plaintext, bs)

	ciphertext := make([]byte, len(plaintext))
	dst := ciphertext
	for len(plaintext) > 0 {
		block.Encrypt(dst, plaintext[:bs])
		plaintext = plaintext[bs:]
		dst = dst[bs:]
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := strings.Repeat(string(byte(padding)), padding)
	return append(data, []byte(padtext)...)
}

// DecryptByAes aes密码解密
func DecryptByAes(encryptedHex string, keyWord string) (string, error) {
	key := []byte(keyWord)
	encrypted, err := base64.StdEncoding.DecodeString(encryptedHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	bs := block.BlockSize()
	if len(encrypted)%bs != 0 {
		return "", errors.New("ciphertext length is not a multiple of the block size")
	}

	plaintext := make([]byte, len(encrypted))
	src := encrypted
	dst := plaintext
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}

	plaintext = pkcs7Unpadding(plaintext)

	return string(plaintext), nil
}

func pkcs7Unpadding(data []byte) []byte {
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}

// EncryptByRsa Rsa密码加签
func EncryptByRsa(txt string, pubKey string) (string, error) {
	block, _ := pem.Decode([]byte(pubKey))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("not RSA public key")
	}
	encryptedBytes, err := rsa.EncryptPKCS1v15(nil, rsaPub, []byte(txt))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

// DecryptByRsa Rsa密码解密
func DecryptByRsa(encryptedTxt string, privateKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the private key")
	}
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	rsaPriv, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not RSA private key")
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedTxt)
	if err != nil {
		return "", err
	}

	decryptedBytes, err := rsa.DecryptPKCS1v15(nil, rsaPriv, encryptedBytes)
	if err != nil {
		return "", err
	}

	return string(decryptedBytes), nil
}
