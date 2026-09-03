// SPDX-License-Identifier: AGPL-3.0-only
package whisper

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
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

func TestHandshakeAndBreathOverTLS(t *testing.T) {
	ts := httptest.NewTLSServer(NewServer())
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
	res, err := client.Word("breath", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Result != "alive" {
		t.Fatalf("breath failed: %+v", res)
	}

	resp, err := client.http.Get(ts.URL + "/whisper/v1/shards")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var shards []Shard
	if err := json.NewDecoder(resp.Body).Decode(&shards); err != nil {
		t.Fatal(err)
	}
	if len(shards) != 1 || shards[0].ID != "shard-test-1" {
		t.Fatalf("unexpected shard listing: %+v", shards)
	}
}
