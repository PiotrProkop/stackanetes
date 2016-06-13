package main

import (
	"fmt"
	"os"

	client "k8s.io/kubernetes/pkg/client/unversioned"
)

type container struct {
	c *client.Client
}

func (cont container) GetType() string {
	return "container"
}

func (cont container) Exists(namespace string, name string) error {

	pod := os.Getenv("POD_NAME")
	if pod == "" {
		Error.Println("No POD_NAME environment variable set")
		os.Exit(1)
	}
	p, err := cont.c.Pods(namespace).Get(pod)
	if err != nil {
		return err
	}
	containers := p.Spec.Containers
	for _, c := range containers {
		if c.Name == name {
			return nil
		}
	}
	return fmt.Errorf("No %s container in %s Pod", name, pod)
}

func (cont container) DepResolved(namespace string, name string) (bool, error) {
	p, err := cont.c.Pods(namespace).Get(os.Getenv("POD_NAME"))
	if err != nil {
		return false, err
	}
	containers := p.Status.ContainerStatuses
	for _, c := range containers {
		if c.Name == name && c.State.Running != nil {
			return true, nil
		}
	}

	return false, nil
}
