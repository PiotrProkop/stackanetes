package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	//For testing purposes
	// restclient "k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	WAITING = "waiting"
	READY   = "ready"
)

var (
	Info  *log.Logger
	Error *log.Logger
)

// This function executes command passed as array of strings
func ExecuteCommandFromAnnotation(command []string) error {
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

	cmd.Run()
	return nil
}

//This function retrives a command section and dependencies section from k8s annotations
func GetAnnotations(annotations map[string]string) (command []string, deps []string) {
	command = strings.Split(annotations["command"], ",")
	deps = strings.Split(annotations["dependencies"], ",")
	if len(deps) == 0 {
		return command, nil
	}

	return command, deps
}

//This function check if a service in given namespace exists
func CheckIfServiceExists(c *client.Client, namespace string, service string) {

	_, err := c.Services(namespace).Get(service)
	if err != nil {
		Error.Println(service, "service doesn't exist in", namespace, "namespace.")
		os.Exit(1)
	}
}

//This function check if given service has at least one endpoint active
func CheckEndpointsAvailabilty(c *client.Client, namespace string, service string) bool {

	e, err := c.Endpoints(namespace).Get(service)
	if err != nil {
		Error.Println(service, "service doesn't exist in", namespace, "namespace.")
		os.Exit(1)
	}
	if len(e.Subsets) == 0 {
		return false
	}
	return true
}

func InitLogger(linfo io.Writer, lerror io.Writer) {
	Info = log.New(linfo, "Entrypoint INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(lerror, "Entrypoint Error: ", log.Ldate|log.Ltime|log.Lshortfile)
}
func main() {
	//Those envs should be set as DownwardAPI http://kubernetes.io/docs/user-guide/downward-api/
	podName := os.Getenv("POD_NAME")
	namespace := os.Getenv("NAMESPACE")
	//Set Logger
	InitLogger(os.Stdout, os.Stderr)
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

	p, err := c.Pods(namespace).Get(podName)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}

	command, deps := GetAnnotations(p.Annotations)

	state := WAITING
	if deps == nil {
		state = READY
	}

	for state == WAITING {

		for i := range deps {
			service := strings.Trim(deps[i], " ")

			CheckIfServiceExists(c, namespace, service)

			if !CheckEndpointsAvailabilty(c, namespace, service) {
				Info.Println(service, " service has no endpoints avaiable -> State waiting")
				state = WAITING
				break
			}

			Info.Println(service, " service has at least one running endpoint. -> State Ready")
			state = READY

		}

	}
	Info.Println("All dependencies resolved")
	err = ExecuteCommandFromAnnotation(command)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}

}
