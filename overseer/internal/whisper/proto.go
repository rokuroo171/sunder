// SPDX-License-Identifier: AGPL-3.0-only
package whisper

// StartResp is the server answer to a handshake start
type StartResp struct {
	Session string `json:"session"`
	EphPub  string `json:"eph_pub"`
	TTLS    int    `json:"ttl_s"`
}

// CompleteReq carries the client half of the key exchange
type CompleteReq struct {
	Session string `json:"session"`
	EphPub  string `json:"eph_pub"`
	Nonce   string `json:"nonce"`
	CT      string `json:"ct"`
}

// Envelope is a sealed payload, hex nonce plus hex ciphertext
type Envelope struct {
	Nonce string `json:"nonce"`
	CT    string `json:"ct"`
}

// Hello is the identity a Shard presents during the handshake
type Hello struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	User     string `json:"user"`
}

// Ack is the server answer that completes the handshake
type Ack struct {
	ShardID   string `json:"shard_id"`
	SessionID string `json:"session_id"`
	CadenceS  int    `json:"cadence_s"`
	Server    string `json:"server"`
	TS        int64  `json:"ts"`
}

// Word is a sealed task for a Shard
type Word struct {
	ShardID string            `json:"shard_id"`
	Word    string            `json:"word"`
	Args    map[string]string `json:"args,omitempty"`
}

// WordResult is the sealed answer to a Word
type WordResult struct {
	Word   string `json:"word"`
	OK     bool   `json:"ok"`
	Result string `json:"result"`
	TS     int64  `json:"ts"`
}
