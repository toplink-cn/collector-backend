package crypt_util

import (
	"collector-backend/db"
	"context"
	"encoding/pem"
	"log"
	"sync"

	"github.com/farmerx/gorsa"
	"github.com/go-redis/redis/v8"
)

var once sync.Once

var internalCryptUtil *CryptUtil

var encryptChunkSize = 501
var decryptChunkSize = 512

type CryptUtil struct {
}

func New() *CryptUtil {
	once.Do(func() {
		internalCryptUtil = &CryptUtil{}

		client := db.GetRedisConnection()
		privateKeyPEM, err := client.Get(context.Background(), "RSAPrivateKeyPem").Result()
		if err == redis.Nil {
			log.Fatal("Private key not found")
		} else if err != nil {
			log.Fatal(err)
		}

		block, _ := pem.Decode([]byte(privateKeyPEM))
		if block == nil || block.Type != "PRIVATE KEY" {
			log.Fatal("Failed to decode private key, ", privateKeyPEM)
		}

		if err := gorsa.RSA.SetPrivateKey(string(privateKeyPEM)); err != nil {
			log.Fatalln(`set private key :`, err)
		}

		publicKeyPEM, err := client.Get(context.Background(), "RSAPublicKeyPem").Result()
		if err == redis.Nil {
			log.Fatal("Public key not found")
		} else if err != nil {
			log.Fatal(err)
		}

		block, _ = pem.Decode([]byte(publicKeyPEM))
		if block == nil || block.Type != "PUBLIC KEY" {
			log.Fatal("Failed to decode public key ,", publicKeyPEM)
		}

		if err := gorsa.RSA.SetPublicKey(string(publicKeyPEM)); err != nil {
			log.Fatalln(`set public key :`, err)
		}
	})

	return internalCryptUtil
}

func (cu *CryptUtil) EncryptViaPub(input []byte) ([]byte, error) {
	var encryptedData []byte
	chunkSize := encryptChunkSize
	for i := 0; i < len(input); i += chunkSize {
		if i+chunkSize > len(input) {
			chunkSize = len(input) - i
		}
		data := input[i : i+chunkSize]
		encrypted, err := gorsa.RSA.PubKeyENCTYPT(data)
		if err != nil {
			return []byte{}, err
		}
		encryptedData = append(encryptedData, encrypted...)
	}
	return encryptedData, nil
}

func (cu *CryptUtil) DecryptViaPub(input []byte) ([]byte, error) {
	decryptedData := []byte{}
	for i := 0; i < len(input); i += decryptChunkSize {
		data := input[i : i+decryptChunkSize]
		decrypted, err := gorsa.RSA.PubKeyDECRYPT(data)
		if err != nil {
			log.Println(data)
			log.Println(string(data))
			return []byte{}, err
		}
		decryptedData = append(decryptedData, decrypted...)
	}
	return decryptedData, nil
}

func (cu *CryptUtil) EncryptViaPrivate(input []byte) ([]byte, error) {
	var encryptedData []byte
	chunkSize := encryptChunkSize
	for i := 0; i < len(input); i += chunkSize {
		if i+chunkSize > len(input) {
			chunkSize = len(input) - i
		}
		data := input[i : i+chunkSize]
		encrypted, err := gorsa.RSA.PriKeyENCTYPT(data)
		if err != nil {
			return []byte{}, err
		}
		encryptedData = append(encryptedData, encrypted...)
	}
	return encryptedData, nil
}

func (cu *CryptUtil) DecryptViaPrivate(input []byte) ([]byte, error) {
	decryptedData := []byte{}
	for i := 0; i < len(input); i += decryptChunkSize {
		data := input[i : i+decryptChunkSize]
		decrypted, err := gorsa.RSA.PriKeyDECRYPT(data)
		if err != nil {
			log.Println(data)
			log.Println(string(data))
			return []byte{}, err
		}
		decryptedData = append(decryptedData, decrypted...)
	}
	return decryptedData, nil
}
