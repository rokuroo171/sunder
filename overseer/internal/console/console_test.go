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
	queued map[string]string
	result whisper.TaskResult
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

func (f *fakeReg) QueueTask(shardID, word string, args map[string]string) (string, error) {
	f.queued = make(map[string]string)
	f.queued["word"] = word
	for k, v := range args {
		f.queued[k] = v
	}
	return "task-1", nil
}

func (f *fakeReg) WaitTaskResult(taskID string, timeout time.Duration) (whisper.TaskResult, error) {
	return f.result, nil
}

func runConsole(t *testing.T, reg Controller, input string) string {
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
	for _, w := range []string{"shards", "grasp", "breath", "tone", "utter", "gaze", "unfold", "pulse", "anatomy"} {
		if !strings.Contains(out, w) {
			t.Fatalf("help missing %s: %q", w, out)
		}
	}
}

func TestUtterRunsOnGraspedShard(t *testing.T) {
	reg := &fakeReg{
		shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())},
		result: whisper.TaskResult{TaskID: "task-1", Word: "utter", OK: true, Result: "hello from the far side"},
	}
	out := runConsole(t, reg, "grasp deadbeef\nutter echo hello\nexit\n")
	if !strings.Contains(out, "hello from the far side") {
		t.Fatalf("missing utter result: %q", out)
	}
	if reg.queued["word"] != "utter" || reg.queued["command"] != "echo hello" {
		t.Fatalf("wrong task queued: %+v", reg.queued)
	}
}

func TestUtterRequiresGrasp(t *testing.T) {
	out := runConsole(t, &fakeReg{}, "utter echo hi\nexit\n")
	if !strings.Contains(out, "grasp a shard first") {
		t.Fatalf("missing grasp guidance: %q", out)
	}
}

func TestUtterTimeoutFlag(t *testing.T) {
	reg := &fakeReg{shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())}}
	out := runConsole(t, reg, "grasp deadbeef\nutter -t 3 sleep 5\nexit\n")
	if reg.queued["t"] != "3" || reg.queued["command"] != "sleep 5" {
		t.Fatalf("wrong utter args: %+v", reg.queued)
	}
	if strings.Contains(out, "usage:") {
		t.Fatalf("timeout flag should parse: %q", out)
	}
}

func TestUnfoldBuildsArgs(t *testing.T) {
	reg := &fakeReg{shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())}}
	out := runConsole(t, reg, "grasp deadbeef\nunfold /etc/hostname -o 2 -n 4 -t hex\nexit\n")
	if reg.queued["word"] != "unfold" ||
		reg.queued["path"] != "/etc/hostname" ||
		reg.queued["o"] != "2" ||
		reg.queued["n"] != "4" ||
		reg.queued["t"] != "hex" {
		t.Fatalf("wrong unfold args: %+v", reg.queued)
	}
	if strings.Contains(out, "usage:") {
		t.Fatalf("unfold flags should parse: %q", out)
	}
}

func TestGazePassesPath(t *testing.T) {
	reg := &fakeReg{shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())}}
	runConsole(t, reg, "grasp deadbeef\ngaze /tmp\nexit\n")
	if reg.queued["word"] != "gaze" || reg.queued["path"] != "/tmp" {
		t.Fatalf("wrong gaze args: %+v", reg.queued)
	}
}

func TestFailedWordResult(t *testing.T) {
	reg := &fakeReg{
		shards: []whisper.Shard{shard("deadbeefcafe", 5, time.Now())},
		result: whisper.TaskResult{TaskID: "task-1", Word: "utter", OK: false, Result: "exit: 1"},
	}
	out := runConsole(t, reg, "grasp deadbeef\nutter false\nexit\n")
	if !strings.Contains(out, "failed: exit: 1") {
		t.Fatalf("missing failure text: %q", out)
	}
}
