package main

import (
	"io/ioutil"
	//"golang.org/x/crypto/ssh"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

    "github.com/eliastor/github-star-cloner/types"
	"github.com/eliastor/github-star-cloner/provider"
	"github.com/eliastor/github-star-cloner/provider/github.com"
	//Plans:
	//"github.com/eliastor/github-star-cloner/provider/gitlab.com"
	//"github.com/eliastor/github-star-cloner/provider/bitbucket.com"
	//"github.com/eliastor/github-star-cloner/provider/gogs"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/file"
	"github.com/heetch/confita/backend/flags"
	
	"github.com/pkg/profile"
)

const (
	configPath = "config.yml"
)

type config struct {
	Github   string              `config:"github"`
	//Gitlab	 string              `config:"gitlab"`
	//Bitbucket	 string              `config:"bitbucket"`
	providers map[string]provider.Provider
	gitAuth map[string]transport.AuthMethod
	Path    string              `config:"path"`
	Workers int                 `config:"workers"`
	Retries int                 `config:"retries"`
	Skip    map[string]struct{} `config:"skip"`
	writeConfig bool `config:"s"`
}

var conf = config{
	Github:   "",
	providers: make(map[string]provider.Provider),
	gitAuth: make(map[string]transport.AuthMethod),
	Workers: 5,
	Path:    `./`,
	Retries: 3,
	Skip: make(map[string]struct{}, 0),
}

type Job struct {
	Args []interface{}
	err error
}

type ExtendedJob struct {
	Job
	Returns []interface{}

	createTime time.Time
	startTime time.Time
	endTime time.Time
	doneJobs chan<- *Job
	errJobs chan<- *Job
	id string
	workerAssigned string
	critical bool
	ctx context.Context
}

func authMatcher(p provider.Authenticator) (transport.AuthMethod){
	var am transport.AuthMethod
	switch p.Name(){
	case "token":
		/*a:= new(githttp.TokenAuth)
		a.Token = p.Value()
		am = a
		*/
		a := new(githttp.BasicAuth)
		a.Username="blahhhhh"
		a.Password=p.Value()
	case "basic":
		a := new(githttp.BasicAuth)
		a.Username = p.Values()[0]
		a.Password = p.Values()[1]
		am = a
	case "key":
		a,_ := gitssh.NewPublicKeys(p.Values()[0], []byte(p.Values()[1]), p.Values()[2])
		am = a
	}
	return am
}

func idealCloner(job *Job) (err error){
	if err != nil {
		return err
	}
	return
}

type Pool struct {
	jobs chan Job
	doneJobs chan Job
	errJobs chan Job
	worker func (context.Context, Job) (error)
	crier func (context.Context, Job)
	fixer func (context.Context, Job)
	ctx context.Context
	cancel context.CancelFunc
	aciveJobs map[string]Job
	activeJobsLock sync.Mutex
	wg sync.WaitGroup
	timeout time.Duration
}

func (p *Pool) Start(workers, criers, fixers uint) {

}

func (p *Pool) AddJob(job Job){

}

func (p *Pool) Stop(){

}

func (p *Pool) Wait(){

}

func (p *Pool) ActiveJobs() []Job{
	return []Job{}
}

func NewPool() Pool{
	return *new(Pool)
}

func cloner(jobs <-chan types.Repository, doneJobs, errJobs chan<- types.Repository, path string) {
	bw := new(strings.Builder)
	for job := range jobs {
		name := job.Name
		activeJobs.Store(name, nil)
		//timeWatchProbe(name)

		bw.WriteString(name)
		bw.WriteString(": ")

		status, err := clone(job.URL, path+name, conf.gitAuth["github"], nil)
		if err != nil {
			//timeWatchProbe(name)
			//bw.WriteString(err.Error())
			//job.output = bw.String()
			fmt.Println(job.Name, err)
			errJobs <- job
		} else {
			//timeWatchProbe(name)
			bw.WriteString(status)
			bw.WriteString(" in ")
			bw.WriteString(timewatchGet(name).String())
			//job.output = bw.String()
			//job.cloned = true
			//job.errored = false

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

func parseConfig(conf *config) (err error) {

	if conf.Github != "" {
		var auth provider.Authenticator

		sep := strings.SplitN(conf.Github, ":", 2)
		method := sep[0]
		value := sep[1]
			
		switch method{
		case "ssh":
			sep :=strings.SplitN(value, ":", 2)
			path := sep[0]
			user:=""
			if len(sep)>1{
				user = sep[1]
			}

			fileinfo, err := os.Stat(path)
			if err != nil {
				return errors.New(fmt.Sprint("Wrong github config: unable to find ssh key",value))
			}
			
			if !fileinfo.IsDir(){
				return errors.New(fmt.Sprint("Wrong github config:", value, "is directory"))
			}
			fp,err := os.Open(path)
			defer fp.Close()
			key, err := ioutil.ReadAll(fp)
			auth = provider.NewAuthKey(user, key, "")
		case "token":
			token := value
			auth = provider.NewAuthToken(token)
		case "basic":
			/*sep :=strings.SplitN(value, ":", 2)
			user := sep[0]
			pass:=""
			if len(sep)>1{
				pass = sep[1]
			}
			*/
		}

		gitconfig := github.Config {
			Ctx: context.TODO(),
			PreferSSH: false,
			Auth: auth,
		}
		prov,err := provider.NewProvider("github", gitconfig)
		if err != nil {
			return err
		}
		conf.providers["github"]=prov
		conf.gitAuth["github"]=authMatcher(auth)
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
}

var activeJobs sync.Map

func main() {
	defer profile.Start(profile.MemProfile).Stop()

	cfgLoader := new(confita.Loader)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		cfgLoader = confita.NewLoader(
			file.NewBackend(configPath),
		)
	} else {
		cfgLoader = confita.NewLoader(
			flags.NewBackend(),
		)
	}
	err := cfgLoader.Load(context.Background(), &conf)
	if err != nil {
		log.Fatal("Failed to load config: ", err.Error())
	}
	if parseConfig(&conf) != nil {
		log.Fatal("Wrong config", err.Error())
	}

	var repos []types.Repository

	if len(conf.providers) ==0 {
		fmt.Println("No credentials provided")
	}else{
		for key := range conf.providers {
			reps, err := conf.providers[key].Starred()
			if err != nil {
				fmt.Println("Error occured during", key, "usage:", err)
				continue
			}else{
				repos = append(repos, reps...)	
			}
		}
	}
	
	if len(repos) == 0 {
		fmt.Println("No repos to operate.")
	}

	erroredrepos := make([]types.Repository, 0)

	wg := new(sync.WaitGroup)

	go func() {
		timeWatcherServe(3 * time.Second)
	}()

	jobs := make(chan types.Repository, 50)
	errJobs := make(chan types.Repository, 50)
	okJobs := make(chan types.Repository, 50)

	for i := 0; i < conf.Workers; i++ {
		go func() {
			wg.Add(1)
			cloner(jobs, okJobs, errJobs, conf.Path)
			wg.Done()
		}()
	}
	fmt.Println(conf.Workers, "workers started")

	go func(jobs chan<- types.Repository, okJobs <-chan types.Repository) {
		listFile, err := os.OpenFile(conf.Path+"repos.lst", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 660)
		defer func() {
			listFile.Sync()
			if err = listFile.Close(); err != nil {
				listFile.Close()
			}
		}()
		for job := range okJobs {
			listFile.WriteString(job.Name)
			listFile.WriteString(": ")
			listFile.WriteString(job.Descrition)
			listFile.WriteString("\n")
			fmt.Println(job.Name)
		}
		return
	}(jobs, okJobs)

	go func(jobs chan<- types.Repository, errJobs <-chan types.Repository, errRepos []types.Repository) {
		if conf.Retries == 0 {
			return
		}
		if errRepos == nil {
			errRepos = make([]types.Repository, 0)
		}
		retries := make(map[string]int, 50)
		for job := range errJobs {
			name := job.Name
			cnt, ok := retries[name]
			if !ok {
				cnt = 2
				retries[name] = cnt
			}
			if cnt <= conf.Retries {
				fmt.Println(name, "Try", cnt, "/", conf.Retries)
				jobs <- job
			} else {
				fmt.Println(name, "failed")
				errRepos = append(errRepos, job)
			}
		}
	}(jobs, errJobs, erroredrepos)

	for _, rep := range repos {
		if _, ok := conf.Skip[rep.Name]; ok {
			fmt.Println(rep.Name, ": Skipped")
			continue
		}

		jobs <- rep
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
			fmt.Println(errrepo.Name)
		}
	}
}
