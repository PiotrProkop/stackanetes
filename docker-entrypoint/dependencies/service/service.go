package service

import (
	"fmt"
	entry "github.com/stackanetes/docker-entrypoint/dependencies"
	"github.com/stackanetes/docker-entrypoint/util/env"
)

type Service struct {
	name string
}

func init() {
	serviceDeps := env.SplitEnvToList(fmt.Sprintf("%sSERVICE", entry.DependencyPrefix))
	if serviceDeps != nil {
		for dep := range serviceDeps {
			entry.Register(NewService(serviceDeps[dep]))
		}
	}
}

func NewService(name string) (s Service) {
	service := Service{name: name}
	return service
}

func (s Service) IsResolved(entrypoint entry.Entrypoint) (bool, error) {
	e, err := entrypoint.Client.Endpoints(entry.Namespace).Get(s.name)
	if err != nil {
		return false, err
	}
	if len(e.Subsets) > 0 {
		return true, nil
	}
	return false, fmt.Errorf("Service %v has no endpoints", s.GetName())
}

func (s Service) GetName() string {
	return s.name
}
