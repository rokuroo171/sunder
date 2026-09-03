// SPDX-License-Identifier: AGPL-3.0-only
package console

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"sunder/overseer/internal/whisper"
)

type fakeReg struct {
	shards []whisper.Shard
}

func (f *fakeReg) ListShards() []whisper.Shard { return f.shards }

func (f *fakeReg) ShardByID(id string) (whisper.Shard, bool) {
	for _, s := range f.shards {
		if s.ID == id {
			return s, true
		}
	}
	return whisper.Shard{}, false
}

func runConsole(t *testing.T, reg Registry, input string) string {
	t.Helper()
	var out bytes.Buffer
	c := New(reg, strings.NewReader(input), &out, nil)
	if err := c.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	return out.String()
}

func shard(id string, cadence int, last time.Time) whisper.Shard {
	return whisper.Shard{
		ID:         id,
		Hostname:   "lab-" + id[:4],
		OS:         "linux",
		Arch:       "aarch64",
		User:       "tester",
		FirstSeen:  last,
		LastBreath: last,
		CadenceS:   cadence,
	}
}

func TestEmptyShardsVerse(t *testing.T) {
	out := runConsole(t, &fakeReg{}, "shards\nexit\n")
	if !strings.Contains(out, "No shards. The blade is whole, for now.") {
		t.Fatalf("missing verse: %q", out)
	}
}

func TestShardsTable(t *testing.T) {
	reg := &fakeReg{shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())}}
	out := runConsole(t, reg, "shards\nexit\n")
	if !strings.Contains(out, "deadbeefcafe") || !strings.Contains(out, "breathing") {
		t.Fatalf("missing table rows: %q", out)
	}
}

func TestGraspAndBreath(t *testing.T) {
	reg := &fakeReg{shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())}}
	out := runConsole(t, reg, "grasp dead\nbreath\nexit\n")
	if !strings.Contains(out, "You have its ear.") {
		t.Fatalf("missing grasp verse: %q", out)
	}
	if !strings.Contains(out, "It breathes, shallow and patient.") {
		t.Fatalf("missing breath verse: %q", out)
	}
}

func TestBreathDark(t *testing.T) {
	reg := &fakeReg{shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now().Add(-10*time.Minute))}}
	out := runConsole(t, reg, "grasp deadbeef\nbreath\nexit\n")
	if !strings.Contains(out, "Silence. The wire is cold.") {
		t.Fatalf("missing dark verse: %q", out)
	}
}

func TestBreathRequiresGrasp(t *testing.T) {
	out := runConsole(t, &fakeReg{}, "breath\nexit\n")
	if !strings.Contains(out, "grasp a shard first") {
		t.Fatalf("missing guidance: %q", out)
	}
}

func TestToneQuietHidesFlavor(t *testing.T) {
	reg := &fakeReg{shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())}}
	out := runConsole(t, reg, "tone quiet\ngrasp deadbeef\nbreath\nexit\n")
	if strings.Contains(out, "You have its ear.") {
		t.Fatalf("quiet leaked flavor: %q", out)
	}
	if !strings.Contains(out, "last breath") {
		t.Fatalf("quiet hid data: %q", out)
	}
}

func TestPassthrough(t *testing.T) {
	out := runConsole(t, &fakeReg{}, "!echo hello-sunder\nexit\n")
	if !strings.Contains(out, "hello-sunder") {
		t.Fatalf("passthrough failed: %q", out)
	}
}

func TestUnknownWord(t *testing.T) {
	out := runConsole(t, &fakeReg{}, "bogus\nexit\n")
	if !strings.Contains(out, "unknown word") {
		t.Fatalf("missing error: %q", out)
	}
}

func TestHelpListsWords(t *testing.T) {
	out := runConsole(t, &fakeReg{}, "help\nexit\n")
	for _, w := range []string{"shards", "grasp", "breath", "tone"} {
		if !strings.Contains(out, w) {
			t.Fatalf("help missing %s: %q", w, out)
		}
	}
}
