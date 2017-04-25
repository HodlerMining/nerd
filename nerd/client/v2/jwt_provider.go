package v2client

import (
	"github.com/pkg/errors"
)

//JWTProvider provides the client with a JWT. An implementation of this interface
//is capable of providing a JWT string to the client. When IsExpired return false
//the in-memory JWT will be used to prevent from calling Retrieve for each API call.
type JWTProvider interface {
	IsExpired() bool
	Retrieve() (string, error)
}

//StaticJWTProvider is a simple JWT provider that always returns the same JWT.
type StaticJWTProvider struct {
	JWT string
}

//NewStaticJWTProvider creates a new StaticJWTProvider for the given jwt.
func NewStaticJWTProvider(jwt string) *StaticJWTProvider {
	return &StaticJWTProvider{jwt}
}

//IsExpired always returns false.
func (s *StaticJWTProvider) IsExpired() bool {
	return false
}

//Retrieve always returns the given jwt.
func (s *StaticJWTProvider) Retrieve() (string, error) {
	return s.JWT, nil
}

//ChainedJWTProvider provides nerdalize credentials based on multiple providers. The given providers are tried in sequential order.
type ChainedJWTProvider struct {
	Providers []JWTProvider
	curr      JWTProvider
}

//NewChainedJWTProvider creates a new chained jwt provider.
func NewChainedJWTProvider(providers ...JWTProvider) *ChainedJWTProvider {
	return &ChainedJWTProvider{
		Providers: providers,
	}
}

// Retrieve returns the credentials value or error if no provider returned
// without error.
//
// If a provider is found it will be cached and any calls to IsExpired()
// will return the expired state of the cached provider.
func (c *ChainedJWTProvider) Retrieve() (string, error) {
	var provErr error
	for _, p := range c.Providers {
		jwt, err := p.Retrieve()
		if err == nil {
			c.curr = p
			return jwt, nil
		}
		provErr = err
	}
	c.curr = nil

	return "", errors.Wrapf(provErr, "could not retreive token from any provider: %v")
}

// IsExpired will returned the expired state of the currently cached provider
// if there is one.  If there is no current provider, true will be returned.
func (c *ChainedJWTProvider) IsExpired() bool {
	if c.curr != nil {
		return c.curr.IsExpired()
	}

	return true
}
