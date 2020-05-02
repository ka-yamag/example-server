package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func respondOk(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if body == nil {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		enc.Encode(body)
	}
}

type status struct {
	Status string `json:"status"`
}

func handlerStatus() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		respondOk(w, &status{
			Status: "ok",
		})
	})
}

func shutdown() {
	fmt.Println("call 1")
}

func gracefullyShutDown(s *http.Server) {
	ctxShutDoen, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("[+] Start shutdown")
	s.Shutdown(ctxShutDoen)
	fmt.Println("[-] End shutdown")
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/v1/status", handlerStatus())

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	srv.RegisterOnShutdown(shutdown)

	srvch := make(chan error, 100)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer close(srvch)
		defer wg.Done()

		fmt.Println("[+] Start server")
		srvch <- srv.ListenAndServe()
		fmt.Println("[-] End server")
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	go func() {
		for {
			select {
			case <-sigs:
				gracefullyShutDown(srv)
				return
			case err := <-srvch:
				if err != nil && err != http.ErrServerClosed {
					gracefullyShutDown(srv)
					log.Fatal(err.Error())
				}
			}
		}
	}()

	wg.Wait()
	fmt.Println("[-] end")
}
