package main

import client "k8s.io/kubernetes/pkg/client/unversioned"

type service struct {
	c *client.Client
}

func (s service) GetType() string {
	return "service"
}

func (s service) Exists(namespace string, name string) error {
	_, err := s.c.Services(namespace).Get(name)
	if err != nil {
		return err
	}
	return nil
}

func (s service) DepResolved(namespace string, name string) (bool, error) {
	e, err := s.c.Endpoints(namespace).Get(name)
	if err != nil {
		return false, err
	}
	if len(e.Subsets) == 0 {
		return false, nil
	}
	return true, nil
}
