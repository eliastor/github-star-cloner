package provider

import "fmt"

const AuthTokenName = "token"

type AuthToken struct {
	token string
}

func (a *AuthToken) Name() string {
	return AuthTokenName
}

func (a *AuthToken) String() string {
	max := len(a.token)
	if max > 6 {
		max = 6
	}
	return fmt.Sprint("<", AuthTokenName, a.token[:max], "...>")
}

func (a *AuthToken) Value() string {
	return a.token
}

func (a *AuthToken) Values() []string {
	return []string{a.token}
}

func NewAuthToken(token string) *AuthToken {
	a := new(AuthToken)
	a.token = token
	return a
}
