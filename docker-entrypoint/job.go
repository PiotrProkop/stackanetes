package main

import

//For testing purposes
//	restclient "k8s.io/kubernetes/pkg/client/restclient"

client "k8s.io/kubernetes/pkg/client/unversioned"

type job struct {
	c *client.Client
}

func (j job) GetType() string {
	return "job"
}

func (j job) Exists(namespace string, name string) error {
	_, err := j.c.ExtensionsClient.Jobs(namespace).Get(name)
	if err != nil {
		return err
	}
	return nil
}

func (j job) DepResolved(namespace string, name string) (bool, error) {
	jo, err := j.c.ExtensionsClient.Jobs(namespace).Get(name)
	if err != nil {
		return false, err
	}
	if jo.Status.Succeeded == 0 {
		return false, nil
	}
	return true, nil
}
