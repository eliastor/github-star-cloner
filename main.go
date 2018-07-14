package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/file"
	"github.com/heetch/confita/backend/flags"
	"golang.org/x/oauth2"
	git "gopkg.in/src-d/go-git.v4"

	"github.com/pkg/profile"
)

const (
	configPath = "./config.yml"
)

var conf = config{
	Token:   "",
	Workers: 5,
	Path:    `./github/`,
	Retries: 5,
}

type repo struct {
	cloneHTTP   string
	cloneSSH    string
	fullName    string
	description string

	output string

	cloned  bool
	errored bool
}

type config struct {
	Token   string              `config:"token,required"`
	Path    string              `config:"path"`
	Workers int                 `config:"workers"`
	Retries int                 `config:"retries"`
	Skip    map[string]struct{} `config:"skip"`
}

//TODO: refactor this mess
func clone(httpURL string, path string, progressWriter io.Writer) (status string, err error) {
	_, err = git.PlainClone(path, false, &git.CloneOptions{
		URL:               httpURL,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          progressWriter,
		NoCheckout:        true,
	})
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			r, err := git.PlainOpen(path) //ErrRepositoryNotExists
			if err != nil {
				return "", err
			}
			err = r.Fetch(&git.FetchOptions{Progress: progressWriter})
			if err != nil {
				if err == git.NoErrAlreadyUpToDate {
					return err.Error(), nil
				} else {
					return "", err
				}
			}
		} else {
			return "", err
		}
	}
	return "Cloned", nil
}

func cloner(jobs <-chan repo, doneJobs, errJobs chan<- repo, path string) {
	bw := new(strings.Builder)
	for job := range jobs {
		name := job.fullName
		activeJobs.Store(name, nil)
		//timeWatchProbe(name)

		bw.WriteString(name)
		bw.WriteString(": ")
		status, err := clone(job.cloneHTTP, path+name, nil)
		if err != nil {
			job.errored = true
			//timeWatchProbe(name)
			bw.WriteString(err.Error())
			job.output = bw.String()
			errJobs <- job
		} else {
			//timeWatchProbe(name)
			bw.WriteString(status)
			bw.WriteString(" in ")
			bw.WriteString(timewatchGet(name).String())
			job.output = bw.String()
			job.cloned = true
			job.errored = false

			doneJobs <- job
		}
		activeJobs.Delete(name)
		bw.Reset()
	}
	return
}

type list struct {
	filename string
	ch       chan string
	listFile *os.File
}

type timeWatch struct {
	watches sync.Map
	ch      chan string
	times   sync.Map
}

type timeWatchTimer struct {
	t       *time.Ticker
	started time.Time
	pt      chan time.Duration
}

var timeWatcher timeWatch

//TODO: reimplement it using time manager
func timeWatcherServe(d time.Duration) {
	timeWatcher.ch = make(chan string)
	for name := range timeWatcher.ch {
		tw := new(timeWatchTimer)
		tw.started = time.Now()
		tw.t = time.NewTicker(d)
		twi, stopit := timeWatcher.watches.LoadOrStore(name, tw)
		tw = twi.(*timeWatchTimer)
		if stopit {
			tw.t.Stop()
			elapsed := time.Since(tw.started)
			timeWatcher.times.Store(name, elapsed)
			go func(tw *timeWatchTimer, name string) {
				sleep := time.After(d)
				select {
				case <-sleep:
				case _, ok := <-tw.pt:
					if ok {
						//close(tw.pt)
					} else {
						return
					}

				}
			}(tw, name)
			go func(tw *timeWatchTimer, elapsed time.Duration) {
				tw.pt <- elapsed
				close(tw.pt)
			}(tw, elapsed)
			//timeWatcher.watches.Store(name, tw)
		} else {
			tw.pt = make(chan time.Duration)
			timeWatcher.watches.Store(name, tw)
			go func(tw *timeWatchTimer) {
				//TODO: refacrot it using done channel
				for range tw.t.C {
					fmt.Println(name + ":" + "still pending ")
				}
			}(tw)
		}

	}
}

func timeWatchProbe(name string) {
	timeWatcher.ch <- name
}

func timewatchGet(name string) (t time.Duration) {
	twi, ok := timeWatcher.watches.Load(name)
	if ok {
		tw := twi.(*timeWatchTimer)
		if tw.pt != nil {
			t, ok := <-tw.pt
			if ok {
				return t
			}
		}
	}

	ti, ok := timeWatcher.times.Load(name)
	if !ok {
		return 0 * time.Second
	}
	t = ti.(time.Duration)
	return
}

func timeWatcherStop() {
	//close all ch in maps

	close(timeWatcher.ch)
}

//TODO: refactor this mess
func getStarredRepos(token string, perPage int) (cloneReps []repo, err error) {
	cloneReps = make([]repo, 0, 50)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	tc := oauth2.NewClient(ctx, ts)

	gclient := github.NewClient(tc)

	opts := new(github.ActivityListStarredOptions)

	opts.PerPage = perPage
	opts.Page = 1

	for {
		repos, resp, err := gclient.Activity.ListStarred(ctx, "", opts)
		if _, ok := err.(*github.RateLimitError); ok || resp.StatusCode != http.StatusOK {
			//hit rate limit
			fmt.Print("*")
			time.Sleep(1 * time.Second)
		}

		if err != nil {
			return nil, err
		}
		opts.Page++
		if len(repos) == 0 {
			break
		}
		for _, srep := range repos {
			rep := *new(repo)
			starredRep := *srep.GetRepository()
			rep.cloneHTTP = starredRep.GetCloneURL()
			rep.cloneSSH = starredRep.GetSSHURL()
			rep.fullName = starredRep.GetFullName()
			rep.description = starredRep.GetDescription()
			fmt.Print(".")
			cloneReps = append(cloneReps, rep)
		}
		if resp.NextPage == 0 {
			break
		}
	}
	return
}

func configCheck(conf *config) (err error) {
	if conf.Token == "" {
		err = errors.New("No Github token specified")
	}
	if conf.Workers < 1 {
		conf.Workers = 1
	} else if conf.Workers > 32 {
		conf.Workers = 32
	}
	conf.Path = filepath.Clean(conf.Path) + "/"
	return
}

func init() {
	conf.Skip = make(map[string]struct{}, 0)
	cfgLoader := new(confita.Loader)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfgLoader = confita.NewLoader(
			file.NewBackend(configPath),
		)
	} else {
		cfgLoader = confita.NewLoader(
			file.NewBackend(configPath),
		)
	}
	cfgLoader = confita.NewLoader(
		flags.NewBackend(),
	)
	err := cfgLoader.Load(context.Background(), &conf)
	if err != nil {
		log.Fatal("Failed to load config: ", err.Error())
	}
	if configCheck(&conf) != nil {
		log.Fatal("Wrong config", err.Error())
	}

}

var activeJobs sync.Map

func main() {
	//defer profile.Start().Stop()
	defer profile.Start(profile.MemProfile).Stop()

	starredrepos, err := getStarredRepos(conf.Token, 50)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println()
	if len(starredrepos) == 0 {
		fmt.Println("Nothing to clone. There is no starred repos")
	}

	erroredrepos := make([]repo, 0)

	wg := new(sync.WaitGroup)

	go func() {
		timeWatcherServe(3 * time.Second)
	}()

	jobs := make(chan repo, 50)
	errJobs := make(chan repo, 50)
	okJobs := make(chan repo, 50)
	for i := 0; i < conf.Workers; i++ {
		go func() {
			wg.Add(1)
			cloner(jobs, okJobs, errJobs, conf.Path)
			wg.Done()
		}()
	}
	fmt.Println(conf.Workers, "workers started")

	go func(jobs chan<- repo, okJobs <-chan repo) {
		listFile, err := os.OpenFile(conf.Path+"repos.lst", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 660)
		defer func() {
			listFile.Sync()
			if err = listFile.Close(); err != nil {
				listFile.Close()
			}
		}()
		for job := range okJobs {
			fmt.Println(job.output)
			listFile.WriteString(job.fullName)
			listFile.WriteString(": ")
			listFile.WriteString(job.description)
			listFile.WriteString("\n")
		}
		return
	}(jobs, okJobs)

	go func(jobs chan<- repo, errJobs <-chan repo, errRepos []repo) {
		if conf.Retries == 0 {
			return
		}
		if errRepos == nil {
			errRepos = make([]repo, 0)
		}
		retries := make(map[string]int, 50)
		for job := range errJobs {
			name := job.fullName
			cnt, ok := retries[name]
			if !ok {
				cnt = 2
				retries[name] = cnt
			}
			if cnt <= conf.Retries {
				fmt.Println(name, "Try", cnt, "/", conf.Retries, ":", job.output)
				jobs <- job
			} else {
				fmt.Println(name, "failed")
				errRepos = append(errRepos, job)
			}
		}
	}(jobs, errJobs, erroredrepos)

	for _, cloneRep := range starredrepos {
		if _, ok := conf.Skip[cloneRep.fullName]; ok {
			fmt.Println(cloneRep.fullName, ": Skipped")
			continue
		}

		jobs <- cloneRep
	}
	fmt.Println("All jobs sended")

	close(jobs)

	/*go func() {
		t := time.NewTicker(1 * time.Second)
		for range t.C {
			sb := strings.Builder{}
			activeJobs.Range(func(key, _ interface{}) bool {
				sb.WriteString(" ")
				sb.WriteString(key.(string))
				return true
			})
			fmt.Println("Active jobs:", sb.String())
		}
	}()*/
	wg.Wait()
	fmt.Println("All jobs done")

	if len(erroredrepos) > 0 {
		fmt.Println("Some repos are not cloned/updated:")
		for _, errrepo := range erroredrepos {
			fmt.Println(errrepo.fullName)
		}
	}
}
