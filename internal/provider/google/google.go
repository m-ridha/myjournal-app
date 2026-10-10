package google

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github/mridha/myjournal-app/internal/provider"

	"golang.org/x/oauth2"
	goauth "golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

const (
	maxCachedTokens = 128
	lockStripes     = 64
	requestTimeout  = 20 * time.Second
)

// Config carries the environment-derived Google OAuth configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	Scopes       []string
}

// Provider is the Google-backed provider used by the real application.
type Provider struct {
	cfg       Config
	endpoint  oauth2.Endpoint
	client    *http.Client
	validate  func(context.Context, string, string) (*idtoken.Payload, error)
	mu        sync.Mutex
	tokens    map[[32]byte]*oauth2.Token
	locks     [lockStripes]chan struct{}
	mutations [lockStripes]chan struct{}
}

func New(cfg Config) *Provider {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "email", "profile", driveScope}
	}
	cfg.Scopes = append([]string(nil), cfg.Scopes...)
	p := &Provider{cfg: cfg, endpoint: goauth.Endpoint, client: &http.Client{Timeout: requestTimeout}, tokens: make(map[[32]byte]*oauth2.Token)}
	validator, err := idtoken.NewValidator(context.Background(), option.WithHTTPClient(p.client))
	p.validate = func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		if err != nil {
			return nil, err
		}
		return validator.Validate(ctx, token, audience)
	}
	for i := range p.locks {
		p.locks[i] = make(chan struct{}, 1)
		p.mutations[i] = make(chan struct{}, 1)
	}
	return p
}

func (p *Provider) ID() string { return provider.GoogleID }

func (p *Provider) Name() string { return "Google" }

func (p *Provider) Begin(_ context.Context, request provider.LoginRequest) (string, error) {
	if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" || request.State == "" || request.Verifier == "" || request.Nonce == "" {
		return "", fmt.Errorf("google OAuth client configuration is missing")
	}

	cfg := p.oauthConfig()

	options := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(request.Verifier),
		oauth2.SetAuthURLParam("nonce", request.Nonce),
	}
	if request.ForceConsent {
		options = append(options, oauth2.SetAuthURLParam("prompt", "consent"))
	}
	return cfg.AuthCodeURL(request.State, options...), nil
}

var _ provider.Provider = (*Provider)(nil)
