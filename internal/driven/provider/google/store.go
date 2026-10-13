package google

import (
	"context"
	"crypto/sha256"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	provider "github/m-ridha/myjournal-app/internal/core/connection"
	"github/m-ridha/myjournal-app/internal/core/journal"
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
	service   *drive.Service
	lock      func(context.Context) (func(), error)
	discovery *operationDiscovery
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
	client := oauth2.NewClient(ctx, tokenSource{p: p, ctx: ctx, refresh: stored.Refresh})
	client.Timeout = requestTimeout
	client.Transport = timingTransport{next: client.Transport}
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, journal.ErrUnavailable
	}
	// ponytail: fixed lock stripes bound memory; unrelated accounts can briefly serialize.
	hash := sha256.Sum256([]byte(c.AccountID))
	return &Store{service: service, lock: func(ctx context.Context) (func(), error) {
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
