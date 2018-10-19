# Github starred repositories cloner
Clone/pull all repositories you have starred on github

At the moment only github hosting supported via token auth.

Project is under develpment, some there could be major changes in flag usage.

Feel free to contact and send pull-request.

## Installation

Go to releases and pick version for your platform

## Build

```
go get github.com/eliastor/github-star-cloner
go build .
```

optionally you can install git-cloner with `go install github.com/eliastor/github-star-cloner`

## Usage
First of all [create personal access token](https://github.com/settings/tokens/new) on GitHub. You don't need to select any additional permissions. BTW you can name this token something like `git-cloner`. Click on "Generate token" and you'll get string like `27f20f373a59d8d0717ec0bfa424189f7df5ce4e`.

Folow to directory where you want store your repositories and do
```
git-cloner -github "token:27f20f373a59d8d0717ec0bfa424189f7df5ce4e"
```

where `27f20f373a59d8d0717ec0bfa424189f7df5ce4e` is your personal acces token.
git-cloner will clone repositories you've starred, if repository already cloned, it will be git-fetched.

> Note that newly cloned repositories clone with submodules without checkout!

## Roadmap

 - add support for your and private github repositories
 - add gitlab support
 - add bitbucket support
 - add local repositories support (look through directory with repositories and make git-fetch)
 - add gogs support
 - add ssh authenication support
 - add http basic authentication
 - daemon for auto clone