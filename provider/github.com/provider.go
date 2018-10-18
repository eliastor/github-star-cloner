package github

import (
	"github.com/eliastor/github-star-cloner/types"
	"github.com/eliastor/github-star-cloner/provider"
	"github.com/google/go-github/v18/github"
	"golang.org/x/oauth2"
	"context"
	"time"
	"net/http"
	"errors"
	"strings"
)

//Errors for convinient usage
var (
	ErrUnsupportedAuth = errors.New("Unsupported Authentification scheme")
)

const (
	perPage = 50
)

//Config for github provider
type Config struct{
	//If there are both http(s) and ssh methods for cloning repo, force ssh usage
	PreferSSH bool
	//Context for github api
	Ctx context.Context
	//Authenticate with specific auth type
	Auth provider.Authenticator
}

var defaultConfig = Config{
	PreferSSH: false,
}

type adp struct {
	client *github.Client
	Config
}

func (p *adp) Starred() (repos []types.Repository, err error){
		opts := new(github.ActivityListStarredOptions)
		opts.PerPage = perPage
		opts.Page = 1
	
		for {
			reps, resp, err := p.client.Activity.ListStarred(p.Ctx, "", opts)
			if _, ok := err.(*github.RateLimitError); ok || resp.StatusCode != http.StatusOK {
				//hit rate limit
				time.Sleep(1 * time.Second)
				select {
				case <-time.After(1*time.Second):
				case <-p.Ctx.Done():
					return repos, nil
				}
			}
	
			if err != nil {
				return nil, err
			}
			opts.Page++
			if len(reps) == 0 {
				break
			}
			for _, srep := range reps {
				starredRep := *srep.GetRepository()
				var url string
				if p.PreferSSH {
					url = starredRep.GetSSHURL()
				} else {
					url= starredRep.GetCloneURL()
				}
				repos = append(repos, types.Repository{URL: url, Name: starredRep.GetFullName(), Descrition: starredRep.GetDescription()})
			}
			if resp.NextPage == 0 {
				break
			}
		}
	return repos, nil
}

func (p *adp) Private() (repos []types.Repository, err error) {
	return repos, nil
}

func factory(config interface{}) (provider.Provider, error) {
	conf, able := config.(Config)
	if !able {
		conf = defaultConfig
	}
	
	p := new(adp)
	p.Config = conf
	p.Auth = conf.Auth
	switch p.Auth.Name(){
	case provider.AuthTokenName:
		token := p.Auth.Value()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		p.client = github.NewClient(oauth2.NewClient(p.Ctx, ts))
	case "basic":
		cred:= strings.Split(p.Auth.Value(), ":")
		for i:=0; i<(len(cred)-2); i++ {
			cred[0] = strings.Join(append(cred[:1], cred[i]), ":")
		}
		basicAuth:= &github.BasicAuthTransport{Username: cred[0], Password: cred[1]}
		p.client = github.NewClient(basicAuth.Client())
	default:
		return nil, ErrUnsupportedAuth
	}

	return p, nil
} 

func init() {
	provider.RegisterProvider("github", factory)
}