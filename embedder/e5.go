package embedder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	_ "embed"
)

//go:embed worker/e5_worker.py
var e5WorkerScript string

// E5Embedder runs a Python STDIO worker for local E5 embeddings.
type E5Embedder struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	dims    int
	tmpFile string
	counter int64
}

// NewE5Embedder spawns the Python E5 worker process.
func NewE5Embedder(modelPath, pythonPath string) (*E5Embedder, error) {
	tmp, err := os.CreateTemp("", "e5_worker_*.py")
	if err != nil {
		return nil, fmt.Errorf("create temp worker script: %w", err)
	}
	if _, err := tmp.WriteString(e5WorkerScript); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return nil, fmt.Errorf("write worker script: %w", err)
	}
	tmp.Close()

	cmd := exec.Command(pythonPath, tmp.Name(), "--model-path", modelPath)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		os.Remove(tmp.Name())
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.Remove(tmp.Name())
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		os.Remove(tmp.Name())
		return nil, fmt.Errorf("start python worker: %w", err)
	}

	e := &E5Embedder{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		tmpFile: tmp.Name(),
	}

	// Read ready message with 120s timeout
	ready, err := e.readReadyMessage()
	if err != nil {
		e.Close()
		return nil, err
	}
	e.dims = ready.Dimensions

	return e, nil
}

func (e *E5Embedder) readReadyMessage() (*e5ReadyMsg, error) {
	type result struct {
		msg *e5ReadyMsg
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var msg e5ReadyMsg
		line, err := e.stdout.ReadBytes('\n')
		if err != nil {
			ch <- result{err: fmt.Errorf("read ready message: %w", err)}
			return
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			ch <- result{err: fmt.Errorf("parse ready message: %w", err)}
			return
		}
		if msg.Status != "ready" {
			ch <- result{err: fmt.Errorf("unexpected status: %s", msg.Status)}
			return
		}
		ch <- result{msg: &msg}
	}()

	select {
	case r := <-ch:
		return r.msg, r.err
	case <-time.After(120 * time.Second):
		return nil, fmt.Errorf("timeout waiting for worker ready (120s)")
	}
}

func (e *E5Embedder) nextID() string {
	return fmt.Sprintf("req-%d", atomic.AddInt64(&e.counter, 1))
}

func (e *E5Embedder) send(req *e5Request) (*e5Response, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	if _, err := e.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write to worker: %w", err)
	}

	line, err := e.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read from worker: %w", err)
	}

	var resp e5Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("worker error: %s", resp.Error)
	}
	return &resp, nil
}

// Embed converts a single text into a vector embedding (query prefix).
func (e *E5Embedder) Embed(_ context.Context, text string) ([]float32, error) {
	resp, err := e.send(&e5Request{
		ID:        e.nextID(),
		Texts:     []string{text},
		InputType: "query",
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embeddings response")
	}
	return float64sToFloat32s(resp.Embeddings[0]), nil
}

// EmbedBatch converts multiple texts into vector embeddings (passage prefix).
func (e *E5Embedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	resp, err := e.send(&e5Request{
		ID:        e.nextID(),
		Texts:     texts,
		InputType: "passage",
	})
	if err != nil {
		return nil, err
	}
	result := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		result[i] = float64sToFloat32s(emb)
	}
	return result, nil
}

// EmbedBatches processes multiple batches sequentially (local GPU).
func (e *E5Embedder) EmbedBatches(ctx context.Context, batches []Batch, progress BatchProgress) ([]BatchResult, error) {
	results := make([]BatchResult, 0, len(batches))
	completedChunks := 0
	totalChunks := 0
	for _, b := range batches {
		totalChunks += b.Size()
	}

	for i, batch := range batches {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		embeddings, err := e.EmbedBatch(ctx, batch.Contents())
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", i, err)
		}

		results = append(results, BatchResult{
			BatchIndex: batch.Index,
			Embeddings: embeddings,
		})

		completedChunks += batch.Size()
		if progress != nil {
			progress(i, len(batches), completedChunks, totalChunks, false, 1, 0)
		}
	}
	return results, nil
}

// Dimensions returns the embedding vector size.
func (e *E5Embedder) Dimensions() int {
	return e.dims
}

// Close shuts down the Python worker process.
func (e *E5Embedder) Close() error {
	e.stdin.Close()

	done := make(chan error, 1)
	go func() { done <- e.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		e.cmd.Process.Kill()
	}

	os.Remove(e.tmpFile)
	return nil
}

func float64sToFloat32s(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
