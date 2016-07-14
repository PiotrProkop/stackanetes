package entrypoint

import (
	"github.com/stackanetes/docker-entrypoint/logger"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"os"
	"sync"
	"time"
)

var (
	Dependencies     []Resolver      // List containing all dependencies to be resolved
	DependencyPrefix = "DEPENDENCY_" //Prefix for env variables
	Namespace        string          //Namespace for containers
	wg               sync.WaitGroup
	INTERVAL         = 2
)

// Object containing k8s client
type Entrypoint struct {
	Client *client.Client
}

//Constructor for entrypoint
func NewEntrypoint() (entry Entrypoint) {
	var err error
	//entry.Client, err = client.NewInCluster()
	config := &restclient.Config{
		Host: "http://10.91.96.110:8080",
	}
	entry.Client, err = client.New(config)
	if err != nil {
		logger.Error.Printf("Creating client failed:%v", err)
		os.Exit(1)
	}
	Namespace = os.Getenv("NAMESPACE")
	if Namespace == "" {
		logger.Error.Print("NAMESPACE env not set")
		os.Exit(1)
	}
	return entry
}

func (e Entrypoint) Resolve() error {
	for _, dep := range Dependencies {
		wg.Add(1)
		go func(dep Resolver) {
			logger.Info.Printf("Resolving %s", dep.GetName())
			var err error
			status := false
			for status == false {
				status, err = dep.IsResolved(e)
				if err != nil {
					logger.Warning.Printf("Resolving dependency for %v failed: %v", dep.GetName(), err)
				}
				time.Sleep(2 * time.Second)
			}
			wg.Done()
			logger.Info.Printf("Dependency %v is resolved", dep.GetName())

		}(dep)
	}
	wg.Wait()
	return nil

}

type Resolver interface {
	//	GetType() string
	IsResolved(entrypoint Entrypoint) (bool, error)
	GetName() string
}

func Register(res Resolver) {
	if res == nil {
		logger.Error.Printf("resolver: could not register nil Resolvable")
		os.Exit(1)
	}
	Dependencies = append(Dependencies, res)
}
