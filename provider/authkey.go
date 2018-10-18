package provider

import "fmt"

const AuthKeyName = "key"

type AuthKey struct {
	user    string
	key     string
	keypass string
}

func (a *AuthKey) Name() string {
	return AuthKeyName
}

func (a *AuthKey) String() string {
	//add fingerprint
	return fmt.Sprint("<ssh key>")
}

func (a *AuthKey) Value() string {
	return a.key
}

func (a *AuthKey) Values() []string {
	return []string{a.user, a.key, a.keypass}
}

func NewAuthKey(user string, key []byte, keypass string) *AuthKey {
	a := new(AuthKey)
	a.user = user
	a.key = string(key)
	a.keypass = keypass
	return a
}
