package jwt

import (
	"crypto/ecdsa"
	"time"

	"github.com/nerdalize/nerd/nerd/client/credentials"
	v2client "github.com/nerdalize/nerd/nerd/client/v2"
	"github.com/nerdalize/nerd/nerd/conf"
	"github.com/nerdalize/nerd/nerd/payload"
	"github.com/pkg/errors"
)

//AuthAPIProvider provides nerdalize credentials by making a request to the nerdalize auth server.
//The UserPassProvider is used to retreive the username and password required to authenticate with the auth server.
type AuthAPIProvider struct {
	*ProviderBasis

	Client *v2client.Auth
	//UserPassProvider is a function that provides the AuthAPIProvider provider with a username and password. This could for example be a function
	//that reads from stdin.
	UserPassProvider func() (string, string, error)
}

//NewAuthAPIProvider creates a new AuthAPIProvider provider.
func NewAuthAPIProvider(pub *ecdsa.PublicKey, userPassProvider func() (string, string, error), c *v2client.Auth) *AuthAPIProvider {
	return &AuthAPIProvider{
		ProviderBasis: &ProviderBasis{
			ExpireWindow: DefaultExpireWindow,
			Pub:          pub,
		},
		UserPassProvider: userPassProvider,
		Client:           c,
	}
}

//Retrieve retrieves the token from the authentication server.
func (p *AuthAPIProvider) Retrieve() (string, error) {
	user, pass, err := p.UserPassProvider()
	if err != nil {
		return "", errors.Wrap(err, "failed to get username or password")
	}
	jwt, err := p.Client.GetToken(user, pass)
	if err != nil {
		if aerr, ok := err.(*payload.AuthError); ok {
			// TODO: Make user facing
			return "", aerr
		}
		return "", errors.Wrap(err, "failed to get nerd jwt for username and password")
	}
	claims, err := credentials.DecodeTokenWithKey(jwt, p.Pub)
	if err != nil {
		return "", errors.Wrapf(err, "failed to retreive claims from nerd jwt '%v'", jwt)
	}
	err = conf.WriteNerdToken(jwt)
	if err != nil {
		return "", errors.Wrap(err, "failed to write nerd jwt to config")
	}
	p.AlwaysValid = claims.ExpiresAt == 0 // if unset
	p.SetExpiration(time.Unix(claims.ExpiresAt, 0))
	return jwt, nil
}
