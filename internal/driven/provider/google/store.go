package google

import (
	"context"
	"crypto/sha256"
	"sync"
	"sync/atomic"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
)

const folderMIME = "application/vnd.google-apps.folder"

const appTag = "myjournal-v1"

const fileFields = "id,name,mimeType,parents,appProperties,createdTime,modifiedTime,md5Checksum,trashed"

const (
	maxContentBytes = 1 << 20
	previewWorkers  = 4
	readAttempts    = 3
)

const (
	propertyApp  = "app"
	propertyRole = "role"
	propertyRoot = "root"
	propertyYear = "year"
	roleRoot     = "root"
	roleEntries  = "entries"
	roleYear     = "year"
	roleEntry    = "entry"
	textMIME     = "text/plain"
)

// Store uses private app properties for folders; names are presentation only.
// Mutation locks coordinate this process, not external Drive clients or replicas.
type Store struct {
	service       *drive.Service
	lock          func(context.Context) (func(), error)
	discovery     *operationDiscovery
	authorization *readAuthorization
}

var (
	_ journal.Store          = (*Store)(nil)
	_ journal.OperationStore = (*Store)(nil)
)

func (p *Provider) Store(ctx context.Context, c provider.Connection) (journal.Store, error) {
	stored, err := decodeCredentials(c)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	client := oauth2.NewClient(ctx, tokenSource{p: p, ctx: ctx, stored: stored})
	client.Timeout = requestTimeout
	client.Transport = timingTransport{next: client.Transport}
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, journal.ErrUnavailable
	}
	// ponytail: fixed lock stripes bound memory; unrelated accounts can briefly serialize.
	hash := sha256.Sum256([]byte(c.AccountID))
	return &Store{service: service, authorization: &readAuthorization{provider: p, stored: stored}, lock: func(ctx context.Context) (func(), error) {
		return acquire(ctx, p.mutations[int(hash[0])%len(p.mutations)])
	}}, nil
}

func acquire(ctx context.Context, lock chan struct{}) (func(), error) {
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// readAuthorization is shared only by views of a single request's Store.
type readAuthorization struct {
	provider *Provider
	stored   credentials
	once     sync.Once
	err      error
}

// Authorize validates this connection before the cache can release account-wide
// data. A fresh refresh-token exchange bypasses cached access tokens once per
// request; subsequent checks still observe successful local disconnects.
func (s *Store) Authorize(ctx context.Context) error {
	a := s.authorization
	if a == nil {
		return ctx.Err()
	}
	if err := a.provider.checkConnection(ctx, a.stored); err != nil {
		return err
	}
	a.once.Do(func() {
		unlock, err := a.provider.lock(ctx, "token:"+a.stored.Refresh)
		if err != nil {
			a.err = err
			return
		}
		defer unlock()
		if a.err = a.provider.checkConnection(ctx, a.stored); a.err == nil {
			_, a.err = a.provider.refreshToken(ctx, a.stored.Refresh)
		}
	})
	if a.err != nil {
		return a.err
	}
	return a.provider.checkConnection(ctx, a.stored)
}

func (p *Provider) checkConnection(ctx context.Context, stored credentials) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cutoff := p.revocationCutoff(stored.Subject).Load()
	if cutoff != 0 && stored.IssuedAt <= cutoff {
		return journal.ErrAuthRequired
	}
	return nil
}

// Fixed stripes bound memory without evicting revocations. A hash collision can
// conservatively require another account to reconnect. Cutoffs are process-local;
// after restart the provider's fresh grant check remains authoritative.
func (p *Provider) revocationCutoff(subject string) *atomic.Int64 {
	hash := sha256.Sum256([]byte(subject))
	index := (uint16(hash[0])<<8 | uint16(hash[1])) % revocationStripes
	return &p.revokedBefore[index]
}
