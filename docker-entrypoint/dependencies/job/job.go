package job

import (
	"fmt"

	entry "github.com/stackanetes/docker-entrypoint/dependencies"
	"github.com/stackanetes/docker-entrypoint/util/env"
)

type Job struct {
	name string
}

func init() {
	jobsDeps := env.SplitEnvToList(fmt.Sprintf("%sJOBS", entry.DependencyPrefix))
	if jobsDeps != nil {
		for dep := range jobsDeps {
			entry.Register(NewJob(jobsDeps[dep]))
		}
	}
}

func NewJob(name string) (s Job) {
	job := Job{name: name}
	return job
}

func (j Job) IsResolved(entrypoint entry.Entrypoint) (bool, error) {
	job, err := entrypoint.Client.ExtensionsClient.Jobs(entry.Namespace).Get(j.name)
	if err != nil {
		return false, err
	}
	if job.Status.Succeeded == 0 {
		return false, fmt.Errorf("Job %v is not completed yet", j.GetName())
	}
	return true, nil
}

func (j Job) GetName() string {
	return j.name
}
