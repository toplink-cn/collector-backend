package crypt_util

import (
	"collector-backend/util"
	"encoding/pem"
	"log"
	"os"
	"sync"

	"github.com/farmerx/gorsa"
)

var once sync.Once

var internalCryptUtil *CryptUtil
var rootDir string

func init() {
	rootDir = util.GetRootDir()
}

type CryptUtil struct {
}

func New() *CryptUtil {
	once.Do(func() {
		internalCryptUtil = &CryptUtil{}

		privateKeyPEM, err := os.ReadFile(rootDir + "/storage/keys/privateKey.pem")
		if err != nil {
			log.Fatal("Private key not found")
		}

		log.Println("privateKeyPEM: ", privateKeyPEM)
		block, _ := pem.Decode(privateKeyPEM)
		log.Println("block: ", block)
		if block == nil || block.Type != "PRIVATE KEY" {
			log.Println("block Type: ", block.Type)
			log.Fatal("Failed to decode private key")
		}

		if err := gorsa.RSA.SetPrivateKey(string(privateKeyPEM)); err != nil {
			log.Fatalln(`set private key :`, err)
		}
	})

	return internalCryptUtil
}

func (cu *CryptUtil) EncryptViaPub(input []byte) ([]byte, error) {
	return gorsa.RSA.PubKeyENCTYPT(input)
}

func (cu *CryptUtil) DecryptViaPub(input []byte) ([]byte, error) {
	return gorsa.RSA.PubKeyDECRYPT(input)
}

func (cu *CryptUtil) EncryptViaPrivate(input []byte) ([]byte, error) {
	return gorsa.RSA.PriKeyENCTYPT(input)
}

func (cu *CryptUtil) DecryptViaPrivate(input []byte) ([]byte, error) {
	return gorsa.RSA.PriKeyDECRYPT(input)
}
