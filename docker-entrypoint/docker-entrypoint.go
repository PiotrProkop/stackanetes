package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	//For testing purposes 
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
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
	fmt.Println("Executing command: ", path)
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
		fmt.Println("service doesn't exists in", namespace, "namespace.")
		os.Exit(1)
	}
}

//This function check if given service has at least one endpoint active 
func CheckEndpointsAvailabilty(c *client.Client, namespace string, service string) bool {

	e, err := c.Endpoints(namespace).Get(service)
	if err != nil {
		fmt.Println("service doesn't exists in", namespace, "namespace.")
		os.Exit(1)
	}
	if len(e.Subsets) == 0 {
		fmt.Println(service, " service has no endpoints avaiable\nState waiting")
		return false
	}
	return true
}
func main() {

	podName := os.Getenv("POD_NAME")
	namespace := os.Getenv("NAMESPACE")

	// Inside k8s POD we need to initialise client with such function
	c, err := client.NewInCluster()
	// For testing purposes uncomment following section and comment out above and fill Host property
	// config := &restclient.Config{
	// 	Host: "http://127.0.0.1:8080",
	// }
	// c, err := client.New(config)
	
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	p, err := c.Pods(namespace).Get(podName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	command, deps := GetAnnotations(p.Annotations)

	state := "waiting"
	if deps == nil {
		state = "ready"
	}

	for state == "waiting" {

		for i := range deps {
			service := strings.Trim(deps[i], " ")

			CheckIfServiceExists(c, namespace, service)

			if !CheckEndpointsAvailabilty(c, namespace, service) {
				state = "waiting"
				break
			}

			fmt.Println(service, " service has at least one running endpoint.")
			state = "ready"

		}

	}
	fmt.Println("All dependencies resolved")
	err = ExecuteCommandFromAnnotation(command)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
