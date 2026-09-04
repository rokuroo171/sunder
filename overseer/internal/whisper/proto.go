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

// Task is one Word queued for a Shard
type Task struct {
	ID   string            `json:"id"`
	Word string            `json:"word"`
	Args map[string]string `json:"args,omitempty"`
}

// TaskResult is the answer a Shard sends back for a finished Task
type TaskResult struct {
	TaskID string `json:"task_id"`
	Word   string `json:"word"`
	OK     bool   `json:"ok"`
	Result string `json:"result"`
	TS     int64  `json:"ts"`
}

// BeatReq is the sealed body of a Shard check in
type BeatReq struct {
	ShardID string      `json:"shard_id"`
	Result  *TaskResult `json:"result,omitempty"`
}

// BeatAck is the sealed answer to a beat
type BeatAck struct {
	OK   bool  `json:"ok"`
	TS   int64 `json:"ts"`
	Task *Task `json:"task,omitempty"`
}

// BeatPost is the outer frame of a beat request
type BeatPost struct {
	ShardID  string   `json:"shard_id"`
	Envelope Envelope `json:"envelope"`
}
