// Onion v3 crypto functions here are inspired by https://github.com/rdkr/oniongen-go

package tor

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base32"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"
)

type OnionV3 struct {
	onionAddress string

	publicKey  []byte
	privateKey []byte

	privateKeyFile []byte
	publicKeyFile  []byte
}

func GenerateOnionV3() (*OnionV3, error) {

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	return GenerateOnionV3FromKeys(publicKey, privateKey)
}

func GenerateOnionV3FromKeys(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) (*OnionV3, error) {
	onionAddress := fmt.Sprintf("%s.onion", encodePublicKey(publicKey))
	privateKeyFile := append([]byte("== ed25519v1-secret: type0 ==\x00\x00\x00"), privateKey[:]...)
	publicKeyFile := append([]byte("== ed25519v1-public: type0 ==\x00\x00\x00"), publicKey...)

	return &OnionV3{
		onionAddress:   onionAddress,
		publicKey:      publicKey,
		privateKey:     privateKey,
		privateKeyFile: privateKeyFile,
		publicKeyFile:  publicKeyFile,
	}, nil
}

func expandPrivateKey(privateKey ed25519.PrivateKey) [64]byte {

	hash := sha512.Sum512(privateKey[:32])
	hash[0] &= 248
	hash[31] &= 127
	hash[31] |= 64
	return hash

}

func encodePublicKey(publicKey ed25519.PublicKey) string {

	// checksum = H(".onion checksum" || pubkey || version)
	var checksumBytes bytes.Buffer
	checksumBytes.Write([]byte(".onion checksum"))
	checksumBytes.Write([]byte(publicKey))
	checksumBytes.Write([]byte{0x03})
	checksum := sha3.Sum256(checksumBytes.Bytes())

	// onion_address = base32(pubkey || checksum || version)
	var onionAddressBytes bytes.Buffer
	onionAddressBytes.Write([]byte(publicKey))
	onionAddressBytes.Write([]byte(checksum[:2]))
	onionAddressBytes.Write([]byte{0x03})
	onionAddress := base32.StdEncoding.EncodeToString(onionAddressBytes.Bytes())

	return strings.ToLower(onionAddress)

}
