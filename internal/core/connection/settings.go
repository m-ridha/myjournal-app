package connection

import "context"

// Settings is a transport-independent account and navigation view.
type Settings struct{ AccountName, ProviderName, FolderURL string }

func (s *Service) Settings(ctx context.Context, current Session) (Settings, error) {
	p, err := s.active(current)
	if err != nil {
		return Settings{}, err
	}
	result := Settings{AccountName: current.AccountEmail, ProviderName: p.Name()}
	if locator, ok := p.(FolderLocator); ok {
		result.FolderURL, err = locator.FolderURL(ctx, current.Connection)
	}
	return result, err
}
