package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	//
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

//
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

func GetAnnotations(annotations map[string]string) (command []string, deps []string) {
	command = strings.Split(annotations["command"], ",")
	deps = strings.Split(annotations["dependencies"], ",")

	return command, deps
}

func CheckIfServiceExists(c *client.Client, namespace string, service string) {

	_, err := c.Services(namespace).Get(service)
	if err != nil {
		fmt.Println("service doesn't exists in", namespace, "namespace.")
		os.Exit(1)
	}
}

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

	// c, err := client.NewInCluster()
	config := &restclient.Config{
		Host: "http://127.0.0.1:8080",
	}

	c, err := client.New(config)
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

	for {
		pass := true
		for i := range deps {
			service := strings.Trim(deps[i], " ")
			if service == "" {
				break
			}

			CheckIfServiceExists(c, namespace, service)
			if !CheckEndpointsAvailabilty(c, namespace, service) {
				pass = false
				break
			}

			fmt.Println(service, " service has at least one running endpoint.")
			pass = true

		}
		if pass == true {
			fmt.Println("All dependencies resolved.")
			break
		}
	}
	err = ExecuteCommandFromAnnotation(command)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
