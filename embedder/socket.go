package embedder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// SocketPath retourne le chemin du socket Unix pour le daemon embed.
func SocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".grepai", "embed.sock")
}

// SocketServer écoute sur un Unix socket et sert les requêtes d'embedding.
type SocketServer struct {
	emb      Embedder
	listener net.Listener
	wg       sync.WaitGroup
}

type socketRequest struct {
	Text      string `json:"text"`
	InputType string `json:"input_type"` // "query" or "passage"
}

type socketResponse struct {
	Vector     []float32 `json:"vector,omitempty"`
	Dimensions int       `json:"dimensions"`
	Error      string    `json:"error,omitempty"`
}

// NewSocketServer crée un serveur socket qui délègue à l'embedder fourni.
func NewSocketServer(emb Embedder) (*SocketServer, error) {
	sockPath := SocketPath()

	// Supprimer un éventuel vieux socket
	os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", sockPath, err)
	}

	return &SocketServer{
		emb:      emb,
		listener: listener,
	}, nil
}

// Serve accepte les connexions et traite les requêtes. Bloquant.
func (s *SocketServer) Serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.listener.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(ctx, conn)
		}()
	}
}

func (s *SocketServer) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var req socketRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeResponse(conn, socketResponse{Error: err.Error()})
			continue
		}

		vec, err := s.emb.Embed(ctx, req.Text)
		if err != nil {
			writeResponse(conn, socketResponse{Error: err.Error()})
			continue
		}

		writeResponse(conn, socketResponse{
			Vector:     vec,
			Dimensions: s.emb.Dimensions(),
		})
	}
}

func writeResponse(conn net.Conn, resp socketResponse) {
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	conn.Write(data)
}

// Close arrête le serveur et nettoie le socket.
func (s *SocketServer) Close() {
	s.listener.Close()
	s.wg.Wait()
	os.Remove(SocketPath())
}

// SocketEmbedder est un client qui se connecte au daemon via Unix socket.
// Implémente l'interface Embedder.
type SocketEmbedder struct {
	conn   net.Conn
	reader *bufio.Reader
	dims   int
	mu     sync.Mutex
}

// NewSocketEmbedder se connecte au daemon socket. Retourne une erreur si le daemon n'est pas actif.
func NewSocketEmbedder() (*SocketEmbedder, error) {
	sockPath := SocketPath()
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("daemon not running (no socket at %s): %w", sockPath, err)
	}

	// Envoyer un ping pour récupérer les dimensions
	se := &SocketEmbedder{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}

	vec, err := se.Embed(context.Background(), "ping")
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("daemon ping failed: %w", err)
	}
	se.dims = len(vec)

	return se, nil
}

func (e *SocketEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	req := socketRequest{Text: text, InputType: "query"}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if _, err := e.conn.Write(data); err != nil {
		return nil, fmt.Errorf("write to daemon: %w", err)
	}

	line, err := e.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read from daemon: %w", err)
	}

	var resp socketResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("daemon error: %s", resp.Error)
	}

	return resp.Vector, nil
}

func (e *SocketEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

func (e *SocketEmbedder) Dimensions() int {
	return e.dims
}

func (e *SocketEmbedder) Close() error {
	return e.conn.Close()
}
