package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
	//For testing purposes
	//"k8s.io/kubernetes/pkg/client/restclient"

	client "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	WAITING  = "waiting"
	READY    = "ready"
	RUNNING  = "Running"
	INTERVAL = 2
)

var (
	//"Info logger""
	Info *log.Logger
	//"Error logger"
	Error *log.Logger
)

type dependency interface {
	Exists(namespace string, name string) error
	DepResolved(namespace string, name string) (bool, error)
	GetType() string
}

func WaitFor(dep dependency, namespace string, names []string) error {
	depState := WAITING
	if names == nil {
		depState = READY
		Info.Println("Container has no ", dep.GetType(), " dependencies")
	}

	for depState == WAITING {

		for _, name := range names {

			name := strings.TrimSpace(name)
			err := dep.Exists(namespace, name)
			if err != nil {
				Info.Println(dep.GetType(), name, " doesn't exists -> State waiting")
				depState = WAITING
				break
			}
			res, err := dep.DepResolved(namespace, name)
			if err != nil {
				Info.Println(dep.GetType(), name, " doesn't exists -> State waiting")
				depState = WAITING
				break
			}
			if !res {
				Info.Println(dep.GetType(), name, " is not ready -> State waiting")
				depState = WAITING
				break
			}

			Info.Println(dep.GetType(), name, " is ready -> State ready")
			depState = READY

		}
		//Sleep before next request
		time.Sleep(INTERVAL * time.Second)
	}
	Info.Println("All", dep.GetType(), "dependencies resolved")
	return nil
}

// This function executes command passed as array of strings
func ExecuteCommand(command []string) error {
	path, err := exec.LookPath(command[0])
	if err != nil {
		return err
	}
	cmd := exec.Cmd{
		Path:   path,
		Args:   command,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	Info.Println("Executing command: ", command)

	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func ConvertEnvToList(env string, s ...string) (out []string) {
	sep := ","
	if len(s) > 0 {
		sep = s[0]
	}
	environ := os.Getenv(env)
	if environ == "" {
		return nil
	}
	out = strings.Split(environ, sep)
	return out
}

func InitLogger(linfo io.Writer, lerror io.Writer) {
	Info = log.New(linfo, "Entrypoint INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(lerror, "Entrypoint Error: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {

	//Set Logger
	InitLogger(os.Stdout, os.Stderr)
	//Those envs should be set as DownwardAPI http://kubernetes.io/docs/user-guide/downward-api/
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		Error.Println(fmt.Errorf("Environment variable NAMESPACE is empty"))
		os.Exit(1)
	}

	// Inside k8s POD we need to initialise client with such function
	c, err := client.NewInCluster()
	// For testing purposes uncomment following section and comment out above and fill Host property
	// config := &restclient.Config{
	// 	Host: "http://127.0.0.1:8080",
	// }
	// c, err := client.New(config)

	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}
	jobs := ConvertEnvToList("JOBS")
	if jobs != nil {
		j := job{c}
		err = WaitFor(j, namespace, jobs)
		if err != nil {
			Error.Println(err)
			os.Exit(1)
		}
	} else {
		Info.Println("No job dependency")
	}

	services := ConvertEnvToList("SERVICES")
	if services != nil {
		s := service{c}
		err = WaitFor(s, namespace, services)
		if err != nil {
			Error.Println(err)
			os.Exit(1)
		}
	} else {
		Info.Println("No service dependency")
	}

	daemons := ConvertEnvToList("DS")
	if daemons != nil {
		d := daemonset{c}
		err := WaitFor(d, namespace, daemons)
		if err != nil {
			Error.Println(err)
			os.Exit(1)
		}
	} else {
		Info.Println("No daemonset dependency")
	}
	containers := ConvertEnvToList("CONTAINERS")
	if containers != nil {
		con := container{c}
		err := WaitFor(con, namespace, containers)
		if err != nil {
			Error.Println(err)
			os.Exit(1)
		}
	} else {
		Info.Println("No container dependency")
	}
	conf, err := NewConfig()
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}
	err = conf.RenderConfigs()
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}
	command := ConvertEnvToList("COMMAND", " ")
	if command != nil {
		err = ExecuteCommand(command)
		if err != nil {
			Error.Println(err)
			os.Exit(1)
		}
	} else {
		Error.Println("No COMMAND specified")
		os.Exit(1)
	}

}
