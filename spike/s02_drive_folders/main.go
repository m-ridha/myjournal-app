package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
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

const folderMIMEType = "application/vnd.google-apps.folder"

type pendingLogin struct {
	verifier string
}

type experiment struct {
	mode           string
	baseURL        string
	runID          string
	rootID         string
	entriesID      string
	yearID         string
	manualFileName string
	clientID       string
	clientSecret   string
	mu             sync.Mutex
	pending        map[string]pendingLogin
}

func main() {
	mode := flag.String("mode", "create", "experiment stage: create, verify, check-trash, recreate-entries, check-visibility, or check-moved-year")
	runID := flag.String("run-id", "", "run ID printed by create mode")
	rootID := flag.String("root-id", "", "root folder ID printed by create mode")
	entriesID := flag.String("entries-id", "", "entries folder ID printed by create mode")
	yearID := flag.String("year-id", "", "year folder ID printed by create mode")
	manualFileName := flag.String("manual-file-name", "", "name of the file created manually in Drive")
	flag.Parse()

	if *mode != "create" && *mode != "verify" && *mode != "check-trash" && *mode != "recreate-entries" && *mode != "check-visibility" && *mode != "check-moved-year" {
		log.Fatal("unsupported S2 mode")
	}
	clientID, clientSecret := googleCredentials()
	baseURL := strings.TrimRight(envOr("APP_BASE_URL", "http://localhost:3030"), "/")
	if *mode == "create" {
		var err error
		*runID, err = newRunID()
		if err != nil {
			log.Fatal("could not create experiment run ID")
		}
	}
	exp := &experiment{
		mode:           *mode,
		baseURL:        baseURL,
		runID:          *runID,
		rootID:         *rootID,
		entriesID:      *entriesID,
		yearID:         *yearID,
		manualFileName: *manualFileName,
		clientID:       clientID,
		clientSecret:   clientSecret,
		pending:        make(map[string]pendingLogin),
	}
	if err := exp.validateFlags(); err != nil {
		log.Fatal(err)
	}

	router := chi.NewRouter()
	router.Get("/login", exp.login)
	router.Get("/auth/google/callback", exp.callback)
	log.Printf("S2 folder experiment listening at %s", baseURL)
	log.Printf("Open %s/login to authorize this test run", baseURL)
	if err := http.ListenAndServe(":3030", router); err != nil {
		log.Fatal("S2 folder experiment server stopped")
	}
}

func googleCredentials() (string, string) {
	clientID, clientSecret := envOr("GOOGLE_CLIENT_ID", ""), envOr("GOOGLE_CLIENT_SECRET", "")
	if clientID == "" || clientSecret == "" {
		log.Fatal("set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET before starting the S2 spike")
	}
	return clientID, clientSecret
}

func (e *experiment) validateFlags() error {
	if e.mode == "create" {
		return nil
	}
	if e.runID == "" || e.rootID == "" || e.entriesID == "" || e.yearID == "" {
		return fmt.Errorf("%s mode requires -run-id, -root-id, -entries-id, and -year-id", e.mode)
	}
	if e.mode == "check-visibility" && e.manualFileName == "" {
		return fmt.Errorf("check-visibility mode requires -manual-file-name")
	}
	return nil
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

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
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
	switch e.mode {
	case "create":
		err = e.createFolders(ctx, driveService, &output)
	case "verify":
		err = e.verifyFolders(ctx, driveService, &output)
	case "check-trash":
		err = e.checkTrashedEntriesFolder(ctx, driveService, &output)
	case "recreate-entries":
		err = e.recreateEntriesPath(ctx, driveService, &output)
	case "check-visibility":
		err = e.checkManualVisibility(ctx, driveService, &output)
	case "check-moved-year":
		err = e.checkMovedYearFolder(ctx, driveService, &output)
	}
	if err != nil {
		log.Printf("S2 experiment failed at mode %s: %+v", e.mode, err)
		http.Error(w, "S2 Drive check failed; see local spike output", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(output.Bytes())
}

func (e *experiment) createFolders(ctx context.Context, service *drive.Service, w io.Writer) error {
	rootName := "myjournal-s2-" + e.runID
	root, err := e.createFolder(ctx, service, rootName, "", map[string]string{"role": "root", "s2Run": e.runID})
	if err != nil {
		return err
	}
	entries, err := e.createFolder(ctx, service, "entries", root.Id, map[string]string{"role": "entries", "s2Run": e.runID})
	if err != nil {
		return err
	}
	year, err := e.createFolder(ctx, service, "2026", entries.Id, map[string]string{"role": "year", "year": "2026", "s2Run": e.runID})
	if err != nil {
		return err
	}
	e.rootID, e.entriesID, e.yearID = root.Id, entries.Id, year.Id
	if err := e.verifyFolders(ctx, service, w); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(w, "S2_CREATE_AND_METADATA_LOOKUP: passed")
	_, _ = fmt.Fprintf(w, "run_id: %s\nroot_id: %s\nentries_id: %s\nyear_id: %s\nroot_name: %s\n", e.runID, root.Id, entries.Id, year.Id, rootName)
	_, _ = fmt.Fprintln(w, "Save these identifiers locally for verify/check-trash/check-visibility modes. They are not credentials.")
	return nil
}

func (e *experiment) createFolder(ctx context.Context, service *drive.Service, name, parentID string, appProperties map[string]string) (*drive.File, error) {
	file := &drive.File{Name: name, MimeType: folderMIMEType, AppProperties: appProperties}
	if parentID != "" {
		file.Parents = []string{parentID}
	}
	return service.Files.Create(file).Fields("id,name,mimeType,parents,appProperties,trashed").Context(ctx).Do()
}

func (e *experiment) verifyFolders(ctx context.Context, service *drive.Service, w io.Writer) error {
	root, err := e.findOne(ctx, service, "root", "", false)
	if err != nil {
		return fmt.Errorf("root appProperties lookup: %w", err)
	}
	entries, err := e.findOne(ctx, service, "entries", e.rootID, false)
	if err != nil {
		return fmt.Errorf("entries appProperties lookup under root: %w", err)
	}
	year, err := e.findOne(ctx, service, "year", e.entriesID, false)
	if err != nil {
		return fmt.Errorf("year appProperties lookup under entries: %w", err)
	}
	if root.Id != e.rootID || entries.Id != e.entriesID || year.Id != e.yearID {
		return fmt.Errorf("metadata query returned an unexpected folder ID")
	}
	if year.AppProperties["year"] != "2026" {
		return fmt.Errorf("year folder appProperties did not contain year=2026")
	}
	_, _ = fmt.Fprintf(w, "metadata_lookup_ids_match: true\nroot_current_name: %s\n", root.Name)
	return nil
}

func (e *experiment) checkTrashedEntriesFolder(ctx context.Context, service *drive.Service, w io.Writer) error {
	active, err := e.findAll(ctx, service, "entries", e.rootID, false)
	if err != nil {
		return err
	}
	trashed, err := e.findAll(ctx, service, "entries", "", true)
	if err != nil {
		return err
	}
	activeAbsent := len(active) == 0
	trashedFound := false
	for _, folder := range trashed {
		if folder.Id == e.entriesID && folder.Trashed {
			trashedFound = true
		}
	}
	if !activeAbsent || !trashedFound {
		return fmt.Errorf("trash-state mismatch: active entries found=%d (want 0), expected trashed folder ID found=%t", len(active), trashedFound)
	}
	_, _ = fmt.Fprintf(w, "entries_absent_from_active_query: true\nentries_found_in_trashed_query: true\n")
	return nil
}

func (e *experiment) recreateEntriesPath(ctx context.Context, service *drive.Service, w io.Writer) error {
	active, err := e.findAll(ctx, service, "entries", e.rootID, false)
	if err != nil {
		return fmt.Errorf("check active entries folder before recreation: %w", err)
	}
	if len(active) != 0 {
		return fmt.Errorf("refusing recreation: found %d active entries folders under test root; trash the old entries folder first", len(active))
	}
	trashed, err := e.findAll(ctx, service, "entries", "", true)
	if err != nil {
		return fmt.Errorf("check trashed entries folder before recreation: %w", err)
	}
	oldFolderTrashed := false
	for _, folder := range trashed {
		if folder.Id == e.entriesID && folder.Trashed {
			oldFolderTrashed = true
			break
		}
	}
	if !oldFolderTrashed {
		return fmt.Errorf("refusing recreation: original entries folder ID was not found in Drive trash")
	}
	oldEntriesID, oldYearID := e.entriesID, e.yearID
	entries, err := e.createFolder(ctx, service, "entries", e.rootID, map[string]string{"role": "entries", "s2Run": e.runID})
	if err != nil {
		return err
	}
	year, err := e.createFolder(ctx, service, "2026", entries.Id, map[string]string{"role": "year", "year": "2026", "s2Run": e.runID})
	if err != nil {
		return err
	}
	e.entriesID, e.yearID = entries.Id, year.Id
	if err := e.verifyFolders(ctx, service, w); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "entries_recreated_with_new_id: %t\nyear_recreated_with_new_id: %t\n", entries.Id != oldEntriesID, year.Id != oldYearID)
	_, _ = fmt.Fprintf(w, "replacement_entries_id: %s\nreplacement_year_id: %s\n", entries.Id, year.Id)
	return nil
}

func (e *experiment) checkManualVisibility(ctx context.Context, service *drive.Service, w io.Writer) error {
	root, err := service.Files.Get(e.rootID).Fields("id,name").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("get app-created root folder: %w", err)
	}
	rootName := root.Name
	rootQuery := "name = '" + escapeQueryValue(rootName) + "' and mimeType = '" + folderMIMEType + "' and trashed = false"
	roots, err := e.list(ctx, service, rootQuery)
	if err != nil {
		return err
	}
	manualFileQuery := "name = '" + escapeQueryValue(e.manualFileName) + "' and '" + escapeQueryValue(e.yearID) + "' in parents and trashed = false"
	manualFiles, err := e.list(ctx, service, manualFileQuery)
	if err != nil {
		return err
	}
	appRootOnly := len(roots) == 1 && roots[0].Id == e.rootID
	manualFileHidden := len(manualFiles) == 0
	_, _ = fmt.Fprintf(w, "visible_root_matches_app_created_root_only: %t\nmanual_file_invisible_to_app: %t\n", appRootOnly, manualFileHidden)
	if !appRootOnly || !manualFileHidden {
		return fmt.Errorf("manual visibility differed from expected drive.file behavior: visible_root_matches_app_created_root_only: %t; manual_file_invisible_to_app: %t", appRootOnly, manualFileHidden)
	}
	return nil
}

func (e *experiment) checkMovedYearFolder(ctx context.Context, service *drive.Service, w io.Writer) error {
	folders, err := e.findAll(ctx, service, "year", "", false)
	if err != nil {
		return err
	}
	found := false
	for _, folder := range folders {
		if folder.Id == e.yearID && folder.AppProperties["year"] == "2026" {
			found = true
		}
	}
	_, _ = fmt.Fprintf(w, "moved_year_folder_found_by_properties: %t\n", found)
	if !found {
		return fmt.Errorf("moved year folder was not found by appProperties")
	}
	return nil
}

func (e *experiment) findOne(ctx context.Context, service *drive.Service, role, parentID string, trashed bool) (*drive.File, error) {
	files, err := e.findAll(ctx, service, role, parentID, trashed)
	if err != nil {
		return nil, err
	}
	if len(files) != 1 {
		return nil, fmt.Errorf("expected one folder with role %s, found %d", role, len(files))
	}
	return files[0], nil
}

func (e *experiment) findAll(ctx context.Context, service *drive.Service, role, parentID string, trashed bool) ([]*drive.File, error) {
	query := "mimeType = '" + folderMIMEType + "' and trashed = " + fmt.Sprint(trashed) +
		" and appProperties has { key='role' and value='" + escapeQueryValue(role) + "' }" +
		" and appProperties has { key='s2Run' and value='" + escapeQueryValue(e.runID) + "' }"
	if role == "year" {
		query += " and appProperties has { key='year' and value='2026' }"
	}
	if parentID != "" {
		query += " and '" + escapeQueryValue(parentID) + "' in parents"
	}
	return e.list(ctx, service, query)
}

func (e *experiment) list(ctx context.Context, service *drive.Service, query string) ([]*drive.File, error) {
	result, err := service.Files.List().Q(query).Spaces("drive").PageSize(100).Fields("files(id,name,mimeType,parents,appProperties,trashed)").Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return result.Files, nil
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

func escapeQueryValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, "'", "\\'")
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
