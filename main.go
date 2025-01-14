package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	data       map[string]string
	mu         sync.Mutex
	requests   int
	stopChan   chan struct{}
	httpServer *http.Server
}

func NewServer() *Server {
	return &Server{
		data:     make(map[string]string),
		stopChan: make(chan struct{}),
	}
}

func (s *Server) handlePostData(w http.ResponseWriter, r *http.Request) {
	var input map[string]string
	s.mu.Lock()
	s.requests++
	s.mu.Unlock()

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	for k, v := range input {
		if _, exists := s.data[k]; exists {
			errMsg := fmt.Sprintf("Duplicate entry for key: %s", k)
			http.Error(w, errMsg, http.StatusBadRequest)
		}
		s.data[k] = v
	}
	s.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleGetData(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests++
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.data); err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
	}
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests++
	count := s.requests
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]int{"requests": count}); err != nil {
		http.Error(w, "Failed to encode stats", http.StatusInternalServerError)
	}
}

func (s *Server) handleDeleteData(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests++

	if _, exists := s.data[key]; exists {
		delete(s.data, key)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Key not found", http.StatusNotFound)
	}
}

func (s *Server) startBackgroundWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			dataSize := len(s.data)
			requestCount := s.requests
			s.mu.Unlock()

			log.Printf("Server Status: %d requests, %d items in database", requestCount, dataSize)
		case <-s.stopChan:
			log.Println("Stopping background worker...")
			return
		}
	}
}

func (s *Server) setupHandler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /data", s.handlePostData)
	mux.HandleFunc("GET /data", s.handleGetData)
	mux.HandleFunc("GET /stats", s.handleGetStats)
	mux.HandleFunc("DELETE /data/{key}", s.handleDeleteData)

	mux.HandleFunc("/stats", s.handleGetStats)

	return mux
}

func (s *Server) shutdown(shutdownErrChan chan<- error) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	sig := <-stop
	fmt.Println("Got signal:", sig)

	close(s.stopChan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Gracefully shutting down...")
	shutdownErrChan <- s.httpServer.Shutdown(ctx) // here, the graceful shutdown is called/invoked
}

func (s *Server) Serve(port int) {
	mux := s.setupHandler()

	s.httpServer = &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	shutdownError := make(chan error)

	// calling shutdown method
	go s.shutdown(shutdownError)

	go s.startBackgroundWorker()

	log.Printf("Server running on http://localhost:%d", port)

	err := s.httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ListenAndServe error: %v", err)
	}

	err = <-shutdownError
	if err != nil {
		log.Fatalf("Graceful shutdown error: %v", err)
	}
}

func main() {
	server := NewServer()
	server.Serve(4000)
}
