// SPDX-License-Identifier: AGPL-3.0-only
package whisper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const (
	salt = "sunder-whisper-v1"
	info = "session-key"
)

// SessionKey is a 32 byte AES-256 key derived from an ECDH exchange
type SessionKey [32]byte

// NewEphemeral generates an X25519 key pair for one handshake
func NewEphemeral() (*ecdh.PrivateKey, error) {
	return ecdh.X25519().GenerateKey(rand.Reader)
}

// PubHex encodes a public key for the wire
func PubHex(pub *ecdh.PublicKey) string {
	return hex.EncodeToString(pub.Bytes())
}

// DeriveSessionKey computes the shared session key with a peer
func DeriveSessionKey(priv *ecdh.PrivateKey, peerPubHex string) (SessionKey, error) {
	var zero SessionKey
	raw, err := hex.DecodeString(peerPubHex)
	if err != nil {
		return zero, err
	}
	pub, err := ecdh.X25519().NewPublicKey(raw)
	if err != nil {
		return zero, err
	}
	shared, err := priv.ECDH(pub)
	if err != nil {
		return zero, err
	}
	key, err := hkdf.Key(sha256.New, shared, []byte(salt), info, len(zero))
	if err != nil {
		return zero, err
	}
	copy(zero[:], key)
	return zero, nil
}

// Seal encrypts plain into an Envelope with a fresh nonce
func Seal(key SessionKey, plain []byte) (Envelope, error) {
	var env Envelope
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return env, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return env, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return env, err
	}
	env.Nonce = hex.EncodeToString(nonce)
	env.CT = hex.EncodeToString(gcm.Seal(nil, nonce, plain, nil))
	return env, nil
}

// Open decrypts an Envelope back into plain
func Open(key SessionKey, env Envelope) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := hex.DecodeString(env.Nonce)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("whisper: bad nonce length")
	}
	ct, err := hex.DecodeString(env.CT)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, nil)
}
