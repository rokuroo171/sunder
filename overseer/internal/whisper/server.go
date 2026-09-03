// SPDX-License-Identifier: AGPL-3.0-only
package whisper

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	ephTTL            = 60 * time.Second
	defaultCadenceSec = 5
)

// Shard is the public record of a connected Wraith
type Shard struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	Arch       string    `json:"arch"`
	User       string    `json:"user"`
	FirstSeen  time.Time `json:"first_seen"`
	LastBreath time.Time `json:"last_breath"`
	CadenceS   int       `json:"cadence_s"`
}

type shardEntry struct {
	shard Shard
	key   SessionKey
}

type ephSession struct {
	key     *ecdh.PrivateKey
	expires time.Time
}

// Server owns the handshake state and the shard registry
type Server struct {
	mu     sync.Mutex
	shards map[string]*shardEntry
	eph    map[string]ephSession
}

// NewServer creates an empty whisper server
func NewServer() *Server {
	return &Server{
		shards: make(map[string]*shardEntry),
		eph:    make(map[string]ephSession),
	}
}

// ServeHTTP routes the whisper v1 endpoints
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/whisper/v1":
		writeJSON(w, http.StatusOK, map[string]string{
			"service": "sunder-overseer",
			"whisper": "v1",
			"verse":   "a shard will come knocking",
		})
	case r.Method == http.MethodGet && r.URL.Path == "/whisper/v1/shards":
		s.handleShards(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/whisper/v1/handshake/start":
		s.handleStart(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/whisper/v1/handshake/complete":
		s.handleComplete(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/whisper/v1/word":
		s.handleWord(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	key, err := NewEphemeral()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	session := randomHex(8)
	s.mu.Lock()
	s.eph[session] = ephSession{key: key, expires: time.Now().Add(ephTTL)}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, StartResp{
		Session: session,
		EphPub:  PubHex(key.PublicKey()),
		TTLS:    int(ephTTL.Seconds()),
	})
}

func (s *Server) handleComplete(w http.ResponseWriter, r *http.Request) {
	var req CompleteReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	s.mu.Lock()
	eph, ok := s.eph[req.Session]
	if ok && time.Now().After(eph.expires) {
		ok = false
	}
	delete(s.eph, req.Session)
	s.mu.Unlock()
	if !ok {
		writeErr(w, http.StatusUnauthorized, errors.New("whisper: stale or unknown handshake session"))
		return
	}
	key, err := DeriveSessionKey(eph.key, req.EphPub)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	plain, err := Open(key, Envelope{Nonce: req.Nonce, CT: req.CT})
	if err != nil {
		writeErr(w, http.StatusUnauthorized, errors.New("whisper: could not open client hello"))
		return
	}
	var hello Hello
	if err := json.Unmarshal(plain, &hello); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if hello.ID == "" {
		hello.ID = randomHex(8)
	}
	now := time.Now()
	ack := Ack{
		ShardID:   hello.ID,
		SessionID: req.Session,
		CadenceS:  defaultCadenceSec,
		Server:    "sunder-overseer",
		TS:        now.Unix(),
	}
	s.mu.Lock()
	s.shards[hello.ID] = &shardEntry{
		shard: Shard{
			ID:         hello.ID,
			SessionID:  req.Session,
			Hostname:   hello.Hostname,
			OS:         hello.OS,
			Arch:       hello.Arch,
			User:       hello.User,
			FirstSeen:  now,
			LastBreath: now,
			CadenceS:   defaultCadenceSec,
		},
		key: key,
	}
	s.mu.Unlock()
	log.Printf("[shard %s] registered: %s/%s user=%s host=%s", hello.ID, hello.OS, hello.Arch, hello.User, hello.Hostname)
	respEnv, err := Seal(key, mustJSON(ack))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, respEnv)
}

type wordRequest struct {
	ShardID  string   `json:"shard_id"`
	Envelope Envelope `json:"envelope"`
}

func (s *Server) handleWord(w http.ResponseWriter, r *http.Request) {
	var req wordRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	s.mu.Lock()
	entry, ok := s.shards[req.ShardID]
	s.mu.Unlock()
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("whisper: unknown shard"))
		return
	}
	plain, err := Open(entry.key, req.Envelope)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, errors.New("whisper: could not open word"))
		return
	}
	var word Word
	if err := json.Unmarshal(plain, &word); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var res WordResult
	switch word.Word {
	case "breath":
		now := time.Now()
		s.mu.Lock()
		entry.shard.LastBreath = now
		s.mu.Unlock()
		res = WordResult{Word: word.Word, OK: true, Result: "alive", TS: now.Unix()}
		log.Printf("[shard %s] breath: alive", word.ShardID)
	default:
		res = WordResult{Word: word.Word, OK: false, Result: "unknown word", TS: time.Now().Unix()}
		log.Printf("[shard %s] unknown word: %s", word.ShardID, word.Word)
	}
	respEnv, err := Seal(entry.key, mustJSON(res))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, respEnv)
}

func (s *Server) handleShards(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	shards := make([]Shard, 0, len(s.shards))
	for _, entry := range s.shards {
		shards = append(shards, entry.shard)
	}
	writeJSON(w, http.StatusOK, shards)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(mustJSON(v))
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
