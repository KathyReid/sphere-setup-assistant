package main

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"encoding/binary"
	"fmt"
)

func encrypt(key []byte, message []byte, iv uint64) (encmess []byte, err error) {
	plainText := message
 
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
 
	//IV needs to be unique, but doesn't have to be secure.
	iv_ := make([]byte, aes.BlockSize)
    binary.LittleEndian.PutUint64(iv_, iv)

    fmt.Printf("IV = %v\n",iv_)

    padded_size := len(plainText)
    if len(plainText) % aes.BlockSize != 0 {
    	padded_size += len(plainText) % aes.BlockSize
    }
    padded := make([]byte, padded_size)
    copy(padded, plainText)

	cipherText := make([]byte, len(padded))
 
	stream := cipher.NewCFBEncrypter(block, iv_)
	stream.XORKeyStream(cipherText, padded)

	return cipherText, nil
}
 
func decrypt(key []byte, securemess []byte, iv uint64) (decodedmess []byte, err error) {
	cipherText := securemess
 
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
 
	if len(cipherText) < aes.BlockSize {
		fmt.Printf("%v is too short, blockSize=%v\n", len(cipherText), aes.BlockSize)
		err = errors.New("Ciphertext block size is too short!")
		return nil, err
	}
 
	//IV needs to be unique, but doesn't have to be secure.
	iv_ := make([]byte, aes.BlockSize)
    binary.LittleEndian.PutUint64(iv_, iv)

    fmt.Printf("IV = %v\n",iv_)
 
	stream := cipher.NewCFBDecrypter(block, iv_)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(cipherText, cipherText)
 
 	return cipherText, nil
}