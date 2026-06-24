package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()

	cache, err := NewCache(cfg.RedisURL)
	if err != nil {
		log.Printf("warning: cache disabled: %v", err)
	}
	if cache == nil {
		log.Println("cache: disabled (REDIS_URL not set)")
	}

	var limiter *RateLimiter
	if cfg.RedisURL != "" {
		limiter = NewRateLimiter(cfg.RedisURL, 120, 30)
		log.Println("rate limiter: enabled (120/min)")
	} else {
		log.Println("rate limiter: disabled (REDIS_URL not set)")
	}

	handler := &apiHandler{
		cache: cache,
		rdap:  NewRDAPClient(cfg.RDAPBaseURL),
		dns:   NewDNSClient(cfg.DoHEndpoint),
		crtsh: NewCrtShClient(cfg.CrtShURL),
	}
	router := newRouter(handler, limiter)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  25 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("domainintel listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}
	log.Println("bye")
}
