package transaction

import (
	"crypto/ed25519"
	"crypto/rand"
	"kto/until"
)

const (
	AddressSize = 47
	KTOPrefix   = "Kto"
)

type Wallet struct {
	PrivateKey string
	Address    string
}

func NewWallet() *Wallet {
	var publicKey, privateKey []byte
	for len(PublicKeyToAddress(publicKey)) != AddressSize {
		publicKey, privateKey, _ = GenKeyPair()
	}

	wallet := &Wallet{
		PrivateKey: until.Encode(privateKey),
		Address:    PublicKeyToAddress(publicKey),
	}
	return wallet
}

func GenKeyPair() ([]byte, []byte, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return []byte(publicKey), []byte(privateKey), nil
}

func PublicKeyToAddress(publicKey []byte) string {
	pubStr := until.Encode(publicKey)
	return KTOPrefix + pubStr
}

func AddressToPublicKey(address string) []byte {
	return until.Decode(address[len(KTOPrefix):])
}
