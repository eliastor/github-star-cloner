package provider

import "fmt"

const AuthBasicName = "basic"

type AuthBasic struct {
	user, pass string
	//otp
}

func (a *AuthBasic) Name() string {
	return AuthTokenName
}

func (a *AuthBasic) String() string {
	return fmt.Sprint("(", a.user, "**** )")
}

func (a *AuthBasic) Value() string {
	return fmt.Sprint(a.user, ":", a.pass)
}

func (a *AuthBasic) Values() []string {
	return []string{a.user, a.pass}
}

func NewAuthBasic(user, pass string) *AuthBasic {
	a := new(AuthBasic)
	a.user = user
	a.pass = pass
	return a
}
