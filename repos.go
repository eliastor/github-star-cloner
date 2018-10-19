package main

import (
	"io"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

func probeLocalRepo(path string) (repo *git.Repository, err error) {

	return
}

func clone(URL string, path string, auth transport.AuthMethod, progressWriter io.Writer) (status string, err error) {

	repo, err := git.PlainOpen(path)
	switch err {
	case git.ErrRepositoryNotExists:
		_, err = git.PlainClone(path, false, &git.CloneOptions{
			URL:               URL,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			Progress:          progressWriter,
			Auth:              auth,
			NoCheckout:        true,
		})
		if err != nil {
			return "", err
		}
	case nil:
		err = repo.Fetch(&git.FetchOptions{Progress: progressWriter})
		switch err {
		case git.NoErrAlreadyUpToDate:
			return err.Error(), nil
		default:
			return "", err
		case nil:
			return "Fetched", nil
		}
	default:
	}
	return "", err
}
