package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"myjournal-app/internal/config"
	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/driven/clock"
)

func TestInstallManifest(t *testing.T) {
	cfg := config.Config{BaseURL: "https://myjournal.ink", SessionKey: "0123456789abcdef0123456789abcdef", CookieSecure: true}
	router := newTestRouter(cfg, provider.NewRegistry(), clock.System{}, nil)

	for _, path := range []string{"/", "/en/", "/id/", "/settings"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, http.StatusOK, response.Code)
			assertManifestLink(t, response.Body.Bytes())
		})
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/static/manifest.webmanifest", nil))
	require.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Header().Get("Content-Type"), "application/manifest+json")

	var manifest struct {
		ID                        string `json:"id"`
		Name                      string `json:"name"`
		ShortName                 string `json:"short_name"`
		StartURL                  string `json:"start_url"`
		Scope                     string `json:"scope"`
		Display                   string `json:"display"`
		PreferRelatedApplications bool   `json:"prefer_related_applications"`
		Icons                     []struct {
			Src     string `json:"src"`
			Sizes   string `json:"sizes"`
			Type    string `json:"type"`
			Purpose string `json:"purpose"`
		} `json:"icons"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &manifest))
	assert.Equal(t, "/", manifest.ID)
	assert.Equal(t, "myjournal", manifest.Name)
	assert.Equal(t, "myjournal", manifest.ShortName)
	assert.Equal(t, "/", manifest.StartURL)
	assert.Equal(t, "/", manifest.Scope)
	assert.Equal(t, "standalone", manifest.Display)
	assert.False(t, manifest.PreferRelatedApplications)
	require.Len(t, manifest.Icons, 2)

	for index, size := range []int{192, 512} {
		icon := manifest.Icons[index]
		t.Run(icon.Sizes, func(t *testing.T) {
			assert.Equal(t, fmt.Sprintf("%dx%d", size, size), icon.Sizes)
			assert.Equal(t, "image/png", icon.Type)
			assert.Equal(t, "any", icon.Purpose)
			iconResponse := httptest.NewRecorder()
			router.ServeHTTP(iconResponse, httptest.NewRequest(http.MethodGet, icon.Src, nil))
			require.Equal(t, http.StatusOK, iconResponse.Code)
			assert.Equal(t, "image/png", iconResponse.Header().Get("Content-Type"))
			image, err := png.DecodeConfig(iconResponse.Body)
			require.NoError(t, err)
			assert.Equal(t, size, image.Width)
			assert.Equal(t, size, image.Height)
		})
	}
}

func assertManifestLink(t *testing.T, body []byte) {
	t.Helper()

	document, err := html.Parse(bytes.NewReader(body))
	require.NoError(t, err)

	var manifestLinkFound bool
	var visit func(*html.Node, bool)
	visit = func(node *html.Node, inHead bool) {
		if node.Type == html.ElementNode && node.Data == "head" {
			inHead = true
		}
		if inHead && node.Type == html.ElementNode && node.Data == "link" {
			var relation, href string
			for _, attribute := range node.Attr {
				switch attribute.Key {
				case "rel":
					relation = attribute.Val
				case "href":
					href = attribute.Val
				}
			}
			if relation == "manifest" && href == "/static/manifest.webmanifest" {
				manifestLinkFound = true
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child, inHead)
		}
	}
	visit(document, false)

	assert.True(t, manifestLinkFound, "head must include the web app manifest link")
}
