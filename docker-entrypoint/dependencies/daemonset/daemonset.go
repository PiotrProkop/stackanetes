package daemonset

import (
	"fmt"
	"os"

	entry "github.com/stackanetes/docker-entrypoint/dependencies"
	"github.com/stackanetes/docker-entrypoint/logger"
	"github.com/stackanetes/docker-entrypoint/util/env"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
)

type Daemonset struct {
	name string
}

func init() {
	daemonsetsDeps := env.SplitEnvToList(fmt.Sprintf("%sDAEMONSET", entry.DependencyPrefix))
	if daemonsetsDeps != nil {
		for dep := range daemonsetsDeps {
			entry.Register(NewDaemonset(daemonsetsDeps[dep]))
		}
	}
}

func NewDaemonset(name string) (d Daemonset) {
	daemonset := Daemonset{name: name}
	return daemonset
}

func (d Daemonset) IsResolved(entrypoint entry.Entrypoint) (bool, error) {
	daemonset, err := entrypoint.Client.ExtensionsClient.DaemonSets(entry.Namespace).Get(d.name)
	if err != nil {
		return false, err
	}
	label := labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels)
	opts := api.ListOptions{LabelSelector: label}
	pods, err := entrypoint.Client.Pods(entry.Namespace).List(opts)
	if err != nil {
		return false, err
	}
	myPodName := os.Getenv("POD_NAME")
	if myPodName == "" {
		logger.Error.Print("Environment variable POD_NAME not set")
		os.Exit(1)

	}
	myPod, err := entrypoint.Client.Pods(entry.Namespace).Get(myPodName)
	if err != nil {
		logger.Error.Printf("Getting POD: %v failed : %v", myPodName, err)
		os.Exit(1)
	}
	myHost := myPod.Status.HostIP

	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" && pod.Status.HostIP == myHost {
			return true, nil
		} else if pod.Status.HostIP != myHost {
			return false, fmt.Errorf("Hostname mismatch: Daemonset %v is on host %v and Pod %v is on host %v", d.name, pod.Status.HostIP, myPodName, myHost)
		}

	}
	return false, nil
}

func (d Daemonset) GetName() string {
	return d.name
}
