package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcusbello/meridian/internal/server"
)


func main() {
	srv := server.NewServer()
	defer srv.DB.Close()
	srv.Logger.Println("Starting server...")

	mux := http.NewServeMux()
	mux.HandleFunc("/buy/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			srv.BuyItemHandler(w, r)
		} else {
			srv.BuyItemPage(w, r)
		}
	})
	mux.HandleFunc("/dashboard", srv.AuthMiddleware(srv.DashboardHandler))
	mux.HandleFunc("/api/listing", srv.AuthMiddleware(srv.AddListingHandler))
	mux.HandleFunc("/add_listing", srv.AuthMiddleware(srv.AddListingForm))
	mux.HandleFunc("/listings/", srv.ListingPage)
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			srv.LoginHandler(w, r)
		} else {
			srv.LoginPage(w, r)
		}
	})
	mux.HandleFunc("/token/validate", srv.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Return 200 if token is valid
	}))
	
	// set up static file serving
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		srv.Logger.Println("Server running on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srv.Logger.Fatalf("Server error: %v", err)
		}
	}()

	for {
		sig := <-signalChan
		switch sig {
		case syscall.SIGHUP:
			srv.Logger.Println("Received SIGHUP (reload requested)")
		case syscall.SIGINT, syscall.SIGTERM:
			srv.Logger.Printf("Received %s — shutting down...", sig)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				srv.Logger.Fatalf("Graceful shutdown failed: %v", err)
			}
			srv.Logger.Println("Server stopped cleanly.")
			return
		default:
			srv.Logger.Printf("Unhandled signal: %v", sig)
		}
	}
}
