package custom_test

import (
	"crypto/sha256"
	"io"
	"math/big"

	secp256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
)

// Custom secp256k1 crypto implementation (simplified) for testing purposes.
// This implementation is OpenSSL-compatible: the public key and signatures are uncompressed.
// (The CometBFT secp256k1 implementation uses compressed keys and signatures.)
// The signing is deterministic - contrary to the OpenSSL implementation.

const (
	PrivateKeySize = 32
	PubKeySize     = 32
)

var options = ed25519.CustomOptions{
	PubKeyName:     "custom/PubKeySomeSecp",
	PrivKeyName:    "custom/PrivKeySomeSecp",
	PrivateKeySize: PrivateKeySize,
	PubKeySize:     PubKeySize,
}

var cache []cacheItem

type MySecpPrivKey []byte

func (MySecpPrivKey) Bytes() []byte {
	panic("implement me")
}

func (p MySecpPrivKey) Sign(msg []byte) ([]byte, error) {
	priv, _ := secp256k1.PrivKeyFromBytes(p)

	sum := sha256.Sum256(msg)
	sig := ecdsa.Sign(priv, sum[:])

	return sig.Serialize(), nil
}

func (p MySecpPrivKey) PubKey() crypto.PubKey {
	_, pubkeyObject := secp256k1.PrivKeyFromBytes(p)
	pk := pubkeyObject.SerializeUncompressed()
	return MySecpPubKey(pk)
}

func (MySecpPrivKey) Type() string {
	return "mysecpprivkey"
}

func (p *MySecpPrivKey) With(privKey ed25519.PrivKey) ed25519.CustomPrivKey {
	*p = []byte(privKey)
	return p
}

func (MySecpPrivKey) GenPrivKey() ed25519.PrivKey {
	var privKeyBytes [PrivateKeySize]byte
	d := new(big.Int)

	for {
		privKeyBytes = [PrivateKeySize]byte{}
		_, err := io.ReadFull(crypto.CReader(), privKeyBytes[:])
		if err != nil {
			panic(err)
		}

		d.SetBytes(privKeyBytes[:])
		// break if we found a valid point (i.e. > 0 and < N == curverOrder)
		isValidFieldElement := 0 < d.Sign() && d.Cmp(secp256k1.S256().N) < 0
		if isValidFieldElement {
			break
		}
	}

	return privKeyBytes[:]
}

func (MySecpPrivKey) GenPrivKeyFromSecret(_ []byte) ed25519.PrivKey {
	panic("implement me")
}

type MySecpPubKey []byte

func (MySecpPubKey) Address() crypto.Address {
	panic("implement me")
}

func (p MySecpPubKey) Bytes() []byte {
	return []byte(p)
}

func (p MySecpPubKey) VerifySignature(msg []byte, sigStr []byte) bool {
	// parse the public key
	pub, err := secp256k1.ParsePubKey(p)
	if err != nil {
		return false
	}

	// parse the signature:
	signature, err := ecdsa.ParseDERSignature(sigStr)
	if err != nil {
		return false
	}

	sum := sha256.Sum256(msg)
	return signature.Verify(sum[:], pub)
}

func (MySecpPubKey) Type() string {
	panic("implement me")
}

func (MySecpPubKey) With(_ ed25519.PubKey) ed25519.CustomPubKey {
	panic("implement me")
}

func (MySecpPubKey) String() string {
	panic("implement me")
}

type MySecpBatchVerifier struct {
	// The secp256p1 library used does not have a batch verifier, so we improvise.
	ed25519.BatchVerifier
}

type cacheItem struct {
	pubKey    crypto.PubKey
	message   []byte
	signature []byte
}

func (MySecpBatchVerifier) Add(key crypto.PubKey, message, signature []byte) error {
	c := cacheItem{key, message, signature}
	cache = append(cache, c)
	return nil
}

func (MySecpBatchVerifier) Verify() (bool, []bool) {
	results := make([]bool, len(cache))
	finalresult := true
	for index, item := range cache {
		results[index] = item.pubKey.VerifySignature(item.message, item.signature)
		if !results[index] {
			finalresult = false
		}
	}
	return finalresult, results
}

func (MySecpBatchVerifier) With(batchVerifier ed25519.BatchVerifier) ed25519.CustomBatchVerifier {
	return MySecpBatchVerifier{
		BatchVerifier: batchVerifier,
	}
}