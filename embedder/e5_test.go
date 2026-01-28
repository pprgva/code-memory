package embedder

import (
	"bufio"
	"context"
	"os/exec"
	"testing"
)

const mockE5Script = `
import json, sys
print(json.dumps({"status":"ready","dimensions":4}), flush=True)
for line in sys.stdin:
    req = json.loads(line)
    n = len(req["texts"])
    embeddings = [[0.1, 0.2, 0.3, 0.4]] * n
    print(json.dumps({"id": req["id"], "embeddings": embeddings, "dimensions": 4}), flush=True)
`

const mockE5ErrorScript = `
import json, sys
print(json.dumps({"status":"ready","dimensions":4}), flush=True)
for line in sys.stdin:
    req = json.loads(line)
    print(json.dumps({"id": req["id"], "error": "mock failure"}), flush=True)
`

func newMockE5(t *testing.T, script string) *E5Embedder {
	t.Helper()

	cmd := exec.Command("python3", "-c", script)
	cmd.Stderr = nil

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock: %v", err)
	}

	e := &E5Embedder{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}

	ready, err := e.readReadyMessage()
	if err != nil {
		e.Close()
		t.Fatalf("ready: %v", err)
	}
	e.dims = ready.Dimensions

	t.Cleanup(func() { e.Close() })
	return e
}

func TestNewE5Embedder(t *testing.T) {
	e := newMockE5(t, mockE5Script)
	if e.Dimensions() != 4 {
		t.Errorf("dimensions = %d, want 4", e.Dimensions())
	}
}

func TestE5Embed(t *testing.T) {
	e := newMockE5(t, mockE5Script)
	vec, err := e.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 4 {
		t.Errorf("len = %d, want 4", len(vec))
	}
}

func TestE5EmbedBatch(t *testing.T) {
	e := newMockE5(t, mockE5Script)
	vecs, err := e.EmbedBatch(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}
	if len(vecs) != 3 {
		t.Errorf("len = %d, want 3", len(vecs))
	}
	for i, v := range vecs {
		if len(v) != 4 {
			t.Errorf("vecs[%d] len = %d, want 4", i, len(v))
		}
	}
}

func TestE5EmbedBatches(t *testing.T) {
	e := newMockE5(t, mockE5Script)
	batches := []Batch{
		{Entries: []BatchEntry{{Content: "x"}, {Content: "y"}}, Index: 0},
		{Entries: []BatchEntry{{Content: "z"}}, Index: 1},
	}
	var called int
	results, err := e.EmbedBatches(context.Background(), batches, func(_, _, _, _ int, _ bool, _, _ int) {
		called++
	})
	if err != nil {
		t.Fatalf("EmbedBatches: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results len = %d, want 2", len(results))
	}
	if called != 2 {
		t.Errorf("progress called %d times, want 2", called)
	}
}

func TestE5Close(t *testing.T) {
	e := newMockE5(t, mockE5Script)
	if err := e.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestE5WorkerError(t *testing.T) {
	e := newMockE5(t, mockE5ErrorScript)
	_, err := e.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error from worker")
	}
}
