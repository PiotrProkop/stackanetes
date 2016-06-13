package main

import (
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	labels "k8s.io/kubernetes/pkg/labels"
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
	for _, pod := range pods.Items {
		if pod.Status.Phase == RUNNING {
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
