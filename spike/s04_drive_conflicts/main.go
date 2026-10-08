package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type pendingLogin struct{ verifier string }
type experiment struct {
	mode, baseURL, clientID, clientSecret, baselinePath string
	mu                                                  sync.Mutex
	pending                                             map[string]pendingLogin
}

func main() {
	mode := flag.String("mode", "run", "run or check-rename")
	baseline := flag.String("baseline", "", "local metadata-only JSON file; required, must not exist in run mode")
	flag.Parse()
	if (*mode != "run" && *mode != "check-rename") || *baseline == "" {
		log.Fatal("use -mode=run or -mode=check-rename with -baseline <local-path>")
	}
	if *mode == "run" {
		if _, err := os.Stat(*baseline); !os.IsNotExist(err) {
			log.Fatal("baseline path must not already exist")
		}
	}
	e := &experiment{mode: *mode, baselinePath: *baseline,
		baseURL:  strings.TrimRight(envOr("APP_BASE_URL", "http://localhost:3030"), "/"),
		clientID: os.Getenv("GOOGLE_CLIENT_ID"), clientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		pending: make(map[string]pendingLogin)}
	if e.clientID == "" || e.clientSecret == "" {
		log.Fatal("set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
	}
	if e.baseURL != "http://localhost:3030" {
		log.Fatal("S4 requires APP_BASE_URL=http://localhost:3030")
	}
	router := chi.NewRouter()
	router.Get("/login", e.login)
	router.Get("/auth/google/callback", e.callback)
	server := &http.Server{Addr: "127.0.0.1:3030", Handler: router,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 70 * time.Second, IdleTimeout: 60 * time.Second}
	log.Print("S4 listening at http://localhost:3030; open /login")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("S4 server stopped (check whether port 3030 is occupied)")
	}
}

func (e *experiment) login(w http.ResponseWriter, r *http.Request) {
	state, verifier, err := newPKCEState()
	if err != nil {
		http.Error(w, "could not start OAuth login", http.StatusInternalServerError)
		return
	}
	e.mu.Lock()
	e.pending[state] = pendingLogin{verifier: verifier}
	e.mu.Unlock()

	cfg := e.oauthConfig()
	authURL := cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challengeFor(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (e *experiment) callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	e.mu.Lock()
	login, ok := e.pending[state]
	delete(e.pending, state)
	e.mu.Unlock()
	if !ok {
		http.Error(w, "OAuth state is invalid or expired", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("error") != "" {
		http.Error(w, "Google authorization was declined", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	token, err := e.oauthConfig().Exchange(ctx, r.URL.Query().Get("code"), oauth2.SetAuthURLParam("code_verifier", login.verifier))
	if err != nil {
		http.Error(w, "OAuth code exchange failed", http.StatusUnauthorized)
		return
	}
	driveService, err := drive.NewService(ctx, option.WithTokenSource(e.oauthConfig().TokenSource(ctx, token)))
	if err != nil {
		http.Error(w, "Drive service could not be created", http.StatusInternalServerError)
		return
	}

	var output bytes.Buffer
	if e.mode == "run" {
		err = e.run(ctx, driveService, &output)
	} else {
		err = e.checkRename(ctx, driveService, &output)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(output.Bytes())
	if err != nil {
		// Do not print external error bodies: they can contain private request data.
		_, _ = fmt.Fprintln(w, "S4_CHECK: incomplete; inspect the last stage above; do not mark S4 passed")
	}
}

func (e *experiment) oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     e.clientID,
		ClientSecret: e.clientSecret,
		RedirectURL:  e.baseURL + "/auth/google/callback",
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/drive.file",
		},
		Endpoint: google.Endpoint,
	}
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func newRunID() (string, error) {
	bytes := make([]byte, 9)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
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
