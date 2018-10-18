package provider

import (
	"errors"
	"fmt"

	"github.com/eliastor/github-star-cloner/types"
)

var (
	ErrProviderNotExists = errors.New("Selected provider doesn't exists")
)

type Provider interface {
	Starred() ([]types.Repository, error)
	Private() ([]types.Repository, error)
	//Corporate() []string
	//Auth() transport.AuthMethod
}

type ProviderFactory func(config interface{}) (Provider, error)

type Authenticator interface {
	Name() string
	fmt.Stringer
	Value() string
	Values() []string
	//Interactive() bool
	//Question() <-chan string
	//Answer(string) error
}

type AuthBasic struct {
	user, pass string
	//otp
}

var providers map[string]ProviderFactory

func init() {
	providers = make(map[string]ProviderFactory, 1)
}

func RegisterProvider(name string, factory ProviderFactory) {
	_, exists := providers[name]
	if !exists {
		providers[name] = factory
	}
}

func NewProvider(name string, config interface{}) (Provider, error) {
	factory, exists := providers[name]
	if !exists {
		return nil, ErrProviderNotExists
	}

	return factory(config)
}
