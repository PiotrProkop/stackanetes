package main

import (
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	labels "k8s.io/kubernetes/pkg/labels"
	"os"
)

type daemonset struct {
	c *client.Client
}

func (d daemonset) GetType() string {
	return "daemonset"
}

func (d daemonset) Exists(namespace string, name string) error {
	_, err := d.c.ExtensionsClient.DaemonSets(namespace).Get(name)
	if err != nil {
		return err
	}
	return nil
}

func (d daemonset) DepResolved(namespace string, name string) (bool, error) {
	pods, err := d.GetDsPodsList(namespace, name)
	if err != nil {
		return false, err
	}
	host, err := d.GetHost(namespace, os.Getenv("POD_NAME"))
	if err != nil {
		return false, err
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == RUNNING && pod.Status.HostIP == host {
			return true, nil
		}

	}
	return false, nil
}

func (d daemonset) GetDsPodsList(namespace string, name string) (*api.PodList, error) {
	daemon, err := d.c.ExtensionsClient.DaemonSets(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	label := labels.SelectorFromSet(daemon.Spec.Selector.MatchLabels)
	opts := api.ListOptions{LabelSelector: label}
	pods, err := d.c.Pods(namespace).List(opts)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (d daemonset) GetHost(namespace string, name string) (host string, err error) {
	pod, err := d.c.Pods(namespace).Get(name)
	if err != nil {
		return "", err
	}
	host = pod.Status.HostIP

	return host, nil

}
