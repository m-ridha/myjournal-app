package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	stateToVerifier = make(map[string]string)
)

func main() {
	if os.Getenv("GOOGLE_CLIENT_ID") == "" || os.Getenv("GOOGLE_CLIENT_SECRET") == "" {
		log.Fatal("set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET before starting the spike")
	}
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3030"
	}

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		state, verifier, err := newPKCEState()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		stateToVerifier[state] = verifier
		cfg := oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/auth/google/callback",
			Scopes: []string{
				"openid",
				"email",
				"profile",
				"https://www.googleapis.com/auth/drive.file",
			},
			Endpoint: google.Endpoint,
		}
		authURL := cfg.AuthCodeURL(state,
			oauth2.AccessTypeOffline,
			oauth2.ApprovalForce,
			oauth2.SetAuthURLParam("code_challenge", challengeFor(verifier)),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
			oauth2.SetAuthURLParam("prompt", "consent"),
		)
		http.Redirect(w, r, authURL, http.StatusFound)
	})
	r.Get("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		verifier, ok := stateToVerifier[state]
		if !ok {
			http.Error(w, "missing OAuth PKCE verifier", http.StatusBadRequest)
			return
		}
		delete(stateToVerifier, state)

		cfg := oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/auth/google/callback",
			Scopes: []string{
				"openid",
				"email",
				"profile",
				"https://www.googleapis.com/auth/drive.file",
			},
			Endpoint: google.Endpoint,
		}
		token, err := cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
		if err != nil {
			http.Error(w, fmt.Sprintf("exchange failed: %v", err), http.StatusUnauthorized)
			return
		}

		ds, err := drive.NewService(ctx, option.WithTokenSource(cfg.TokenSource(ctx, token)))
		if err != nil {
			http.Error(w, fmt.Sprintf("drive service failed: %v", err), http.StatusInternalServerError)
			return
		}
		files, err := ds.Files.List().PageSize(5).Fields("files(id,name,parents)").Do()
		if err != nil {
			http.Error(w, fmt.Sprintf("drive list failed: %v", err), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprintf(w, "token acquired, files: %d\n", len(files.Files))
		for _, f := range files.Files {
			_, _ = fmt.Fprintf(w, "- %s\n", f.Name)
		}
	})

	log.Printf("Google OAuth spike running at %s", baseURL)
	log.Printf("Open %s/login to start PKCE sign-in", baseURL)
	if err := http.ListenAndServe(":3030", r); err != nil {
		log.Fatal(err)
	}
}

func newPKCEState() (string, string, error) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", "", err
	}
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", err
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	return state, verifier, nil
}

func challengeFor(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
