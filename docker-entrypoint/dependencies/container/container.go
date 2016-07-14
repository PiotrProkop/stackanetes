package container

import (
	"fmt"
	entry "github.com/stackanetes/docker-entrypoint/dependencies"
	"github.com/stackanetes/docker-entrypoint/util/env"
)

type Container struct {
	name string
}

func init() {
	containerDeps := env.SplitEnvToList(fmt.Sprintf("%sCONTAINER", entry.DependencyPrefix))
	if containerDeps != nil {
		for dep := range containerDeps {
			entry.Register(NewContainer(containerDeps[dep]))
		}
	}
}

func NewContainer(name string) (s Container) {
	container := Container{name: name}
	return container
}

func (c Container) IsResolved(entrypoint entry.Entrypoint) (bool, error) {
	myPodName := os.Getenv("POD_NAME")
	if myPodName == "" {
		return false, fmt.Errorf("Environment variable POD_NAME not set")
	}
	pod, err := entrypoint.Client.Pods(entry.Namespace).Get(c.name)
	if err != nil {
		return false, err
	}
	containers := pod.Status.ContainerStatuses
	for _, container := range containers {
		if container.Name && container.State.Running != nil {
			return true, nil
		}
	}
	return false, nil
}

func (c Container) GetName() string {
	return c.name
}
