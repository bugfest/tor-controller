/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Onion v3 crypto functions here are inspired by https://github.com/rdkr/oniongen-go

package tor

import (
	"fmt"
	"strings"

	// We use "github.com/cretz/bine/torutil/ed25519" instaad of "crypto/ed25519"
	// More info: https://github.com/cathugger/mkp224o/issues/53#issuecomment-874621551
	// An ed25519 key starts out as a 32 byte seed. This seed is hashed with SHA512 to
	// produce 64 bytes (a couple of bits are flipped too). The first 32 bytes of these
	// are used to generate the public key (which is also 32 bytes), and the last 32 bytes
	// are used in the generation of the signature.
	// The Golang private key format is the 32 byte seed concatenated with the 32 byte
	// public key. The private keys in the Bittorrent document you are using are the 64
	// byte result of the hash (or possibly just 64 random bytes that are used the same
	// way as the hash result).

	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"

	torutil "github.com/cretz/bine/torutil"
	ed25519 "github.com/cretz/bine/torutil/ed25519"
)

const (
	onionBalanceSecretVolume = "ob-secret"
	privateKeyVolume         = "private-key"
	torConfigVolume          = "tor-config"
	torConfigExtraVolume     = "tor-config-extra"
	obConfigVolume           = "ob-config"
	onionBalanceConfigVolume = "onionbalance-config"
)

type OnionV3 struct {
	onionAddress string

	publicKey  []byte
	privateKey []byte

	privateKeyFile []byte
	publicKeyFile  []byte
}

func GenerateOnionV3() (*OnionV3, error) {

	k, err := ed25519.GenerateKey(nil)
	publicKey := k.PrivateKey().KeyPair().PublicKey()
	privateKey := k.PrivateKey().KeyPair().PrivateKey()

	if err != nil {
		return nil, err
	}

	return GenerateOnionV3FromKeys(publicKey, privateKey)
}

func GenerateOnionV3FromKeys(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) (*OnionV3, error) {
	// onionAddress := fmt.Sprintf("%s.onion", encodePublicKey(publicKey))
	onionAddress := fmt.Sprintf("%s.onion", torutil.OnionServiceIDFromV3PublicKey(publicKey))
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

// func expandSecretKey(privateKey ed25519.PrivateKey) [64]byte {
// 	hash := sha512.Sum512(privateKey[:32])
// 	hash[0] &= 248
// 	hash[31] &= 127
// 	hash[31] |= 64
// 	return hash
// }

// func encodePublicKey(publicKey ed25519.PublicKey) string {

// 	// checksum = H(".onion checksum" || pubkey || version)
// 	var checksumBytes bytes.Buffer
// 	checksumBytes.Write([]byte(".onion checksum"))
// 	checksumBytes.Write([]byte(publicKey))
// 	checksumBytes.Write([]byte{0x03})
// 	checksum := sha3.Sum256(checksumBytes.Bytes())

// 	// onion_address = base32(pubkey || checksum || version)
// 	var onionAddressBytes bytes.Buffer
// 	onionAddressBytes.Write([]byte(publicKey))
// 	onionAddressBytes.Write([]byte(checksum[:2]))
// 	onionAddressBytes.Write([]byte{0x03})
// 	onionAddress := base32.StdEncoding.EncodeToString(onionAddressBytes.Bytes())

// 	return strings.ToLower(onionAddress)

// }

// Source: https://gitlab.torproject.org/tpo/core/tor/-/blob/main/src/app/main/main.c#L781
// static void
// do_hash_password(void) {
//   char output[256];
//   char key[S2K_RFC2440_SPECIFIER_LEN+DIGEST_LEN];
//   crypto_rand(key, S2K_RFC2440_SPECIFIER_LEN-1);
//   key[S2K_RFC2440_SPECIFIER_LEN-1] = (uint8_t)96; /* Hash 64 K of data. */
//   secret_to_key_rfc2440(key+S2K_RFC2440_SPECIFIER_LEN, DIGEST_LEN,
//                 get_options()->command_arg, strlen(get_options()->command_arg),
//                 key);
//   base16_encode(output, sizeof(output), key, sizeof(key));
//   printf("16:%s\n",output);
// }

// Source: https://gitlab.torproject.org/tpo/core/tor/-/blob/main/src/lib/crypt_ops/crypto_s2k.h#L21
// /** Length of RFC2440-style S2K specifier: the first 8 bytes are a salt, the
//  * 9th describes how much iteration to do. */
//  #define S2K_RFC2440_SPECIFIER_LEN 9
//  void secret_to_key_rfc2440(
// 	char *key_out, size_t key_out_len, const char *secret,
// 	size_t secret_len, const char *s2k_specifier);

// Source: https://gitlab.torproject.org/tpo/core/tor/-/blob/main/src/lib/defs/digest_sizes.h#L20
// #define DIGEST_LEN 20

func doHashPassword(in string) (string, error) {
	const OUTPUT_LEN = 256
	const S2K_RFC2440_SPECIFIER_LEN = 9
	const DIGEST_LEN = 20
	const ITERATIONS = 96
	// 1) Generate S2K_RFC2440_SPECIFIER_LEN-1 random bytes
	// 2) Set last key byte to 96
	//
	// out := make([]byte, DIGEST_LEN)
	salt := make([]byte, S2K_RFC2440_SPECIFIER_LEN-1)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	// Inspired by: https://stackoverflow.com/questions/48054399/get-the-hashed-tor-password-automated-in-python
	EXPBIAS := 6
	c := ITERATIONS
	count := (16 + (c & 15)) << ((c >> 4) + EXPBIAS)
	d := sha1.New()

	inb := []byte(in)
	tmp := append(salt[:S2K_RFC2440_SPECIFIER_LEN-1], inb...)
	slen := len(tmp)

	for count > 0 {
		if count > slen {
			d.Write(tmp)
			count -= slen
		} else {
			d.Write(tmp[:count])
			count = 0
		}
	}
	return fmt.Sprintf("16:%s%s%s",
		strings.ToUpper((hex.EncodeToString(salt))),
		strings.ToUpper((hex.EncodeToString([]byte{ITERATIONS}))),
		strings.ToUpper((hex.EncodeToString(d.Sum(nil)))),
	), nil
}
