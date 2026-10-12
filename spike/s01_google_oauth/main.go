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
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"

	"github/mridha/myjournal-app/spike/internal/loginstate"
)

var pendingLogins loginstate.Store

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
		if !pendingLogins.Add(state, verifier, time.Now()) {
			http.Error(w, "Too many pending logins; retry later", http.StatusTooManyRequests)
			return
		}
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
		authOptions := []oauth2.AuthCodeOption{
			oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("code_challenge", challengeFor(verifier)),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		}
		if r.URL.Query().Get("force_consent") == "1" {
			authOptions = append(authOptions, oauth2.ApprovalForce, oauth2.SetAuthURLParam("prompt", "consent"))
		}
		authURL := cfg.AuthCodeURL(state, authOptions...)
		http.Redirect(w, r, authURL, http.StatusFound)
	})
	r.Get("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		verifier, ok := pendingLogins.Take(state, time.Now())
		if !ok {
			http.Error(w, "OAuth state is invalid or expired", http.StatusBadRequest)
			return
		}

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
			http.Error(w, "OAuth token exchange failed", http.StatusUnauthorized)
			return
		}
		refreshTokenPresent := token.RefreshToken != ""
		idToken, _ := token.Extra("id_token").(string)
		idTokenPresent := idToken != ""
		idTokenValid := false
		subjectPresent := false
		emailPresent := false
		if idTokenPresent {
			payload, err := idtoken.Validate(ctx, idToken, cfg.ClientID)
			if err == nil {
				idTokenValid = true
				subjectPresent = payload.Subject != ""
				email, ok := payload.Claims["email"].(string)
				emailPresent = ok && email != ""
			}
		}

		ds, err := drive.NewService(ctx, option.WithTokenSource(cfg.TokenSource(ctx, token)))
		if err != nil {
			http.Error(w, "Drive service could not be created", http.StatusInternalServerError)
			return
		}
		files, err := ds.Files.List().PageSize(5).Fields("files(id,name,parents)").Do()
		if err != nil {
			http.Error(w, "Drive file listing failed", http.StatusInternalServerError)
			return
		}

		refreshRenewalAttempted := false
		refreshRenewalSucceeded := false
		refreshDriveListSucceeded := false
		if refreshTokenPresent {
			refreshRenewalAttempted = true
			expired := *token
			expired.Expiry = time.Now().Add(-time.Minute)
			refreshed, err := cfg.TokenSource(ctx, &expired).Token()
			refreshRenewalSucceeded = err == nil && refreshed.AccessToken != ""
			if refreshRenewalSucceeded {
				refreshedDrive, serviceErr := drive.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(refreshed)))
				if serviceErr == nil {
					_, listErr := refreshedDrive.Files.List().PageSize(1).Fields("files(id)").Do()
					refreshDriveListSucceeded = listErr == nil
				}
			}
		}

		_, _ = fmt.Fprintf(w, "OAuth callback succeeded\n")
		_, _ = fmt.Fprintf(w, "drive_list_succeeded: true\n")
		_, _ = fmt.Fprintf(w, "drive_files_visible: %d\n", len(files.Files))
		_, _ = fmt.Fprintf(w, "refresh_token_present: %t\n", refreshTokenPresent)
		_, _ = fmt.Fprintf(w, "refresh_renewal_attempted: %t\n", refreshRenewalAttempted)
		_, _ = fmt.Fprintf(w, "refresh_renewal_succeeded: %t\n", refreshRenewalSucceeded)
		_, _ = fmt.Fprintf(w, "refresh_drive_list_succeeded: %t\n", refreshDriveListSucceeded)
		_, _ = fmt.Fprintf(w, "id_token_present: %t\n", idTokenPresent)
		_, _ = fmt.Fprintf(w, "id_token_valid: %t\n", idTokenValid)
		_, _ = fmt.Fprintf(w, "subject_claim_present: %t\n", subjectPresent)
		_, _ = fmt.Fprintf(w, "email_claim_present: %t\n", emailPresent)
	})

	log.Printf("Google OAuth spike running at %s", baseURL)
	log.Printf("Open %s/login to start PKCE sign-in", baseURL)
	if err := http.ListenAndServe("127.0.0.1:3030", r); err != nil {
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
