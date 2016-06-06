package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	//For testing purposes
	//	restclient "k8s.io/kubernetes/pkg/client/restclient"

	client "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	WAITING = "waiting"
	READY   = "ready"
)

var (
	//"Info logger""
	Info *log.Logger
	//"Error logger"
	Error *log.Logger
)

// This function executes command passed as array of strings
func ExecuteCommand(command []string) error {
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
	Info.Println("Executing command: ", command)

	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

//This function retrives a command section and dependencies section from k8s annotations
func GetAnnotations(annotations map[string]string, key string, s ...string) (annotation []string) {
	sep := ","
	if len(s) > 0 {
		sep = s[0]
	}
	annotation = strings.Split(annotations[key], sep)
	if len(annotation) == 0 || annotation[0] == "" {
		return nil
	}

	return annotation
}

func GetEnv(env string, s ...string) (out []string) {
	sep := ","
	if len(s) > 0 {
		sep = s[0]
	}
	out = strings.Split(os.Getenv(env), sep)
	if len(out) == 0 || out[0] == "" {
		return nil
	}
	return out
}

//This function check if a service in given namespace exists
func CheckIfServiceExists(c *client.Client, namespace string, service string) error {

	_, err := c.Services(namespace).Get(service)
	if err != nil {
		return err
	}
	return nil
}

func CheckIfJobExists(c *client.Client, namespace string, job string) error {

	_, err := c.ExtensionsClient.Jobs(namespace).Get(job)
	if err != nil {
		return err
	}
	return nil
}

//This function check if given service has at least one endpoint active
func CheckEndpointsAvailabilty(c *client.Client, namespace string, service string) (bool, error) {

	e, err := c.Endpoints(namespace).Get(service)
	if err != nil {
		return false, err
	}
	if len(e.Subsets) == 0 {
		return false, nil
	}
	return true, nil
}
func IsJobComplete(c *client.Client, namespace string, job string) (bool, error) {
	j, err := c.ExtensionsClient.Jobs(namespace).Get(job)
	if err != nil {
		return false, err
	}
	if j.Status.Succeeded == 0 {
		return false, nil
	}
	return true, nil

}

//"GetIpFromInterface return always first ip from interface"
func GetIpFromInterface(iface string) (string, error) {

	i, err := net.InterfaceByName(iface)
	if err != nil {
		return "", err
	}
	addr, err := i.Addrs()
	if err != nil || len(addr) == 0 {
		return "", err
	}
	return strings.Split(addr[0].String(), "/")[0], nil
}

func RenderConfigWithIP(config string) error {

	err := os.MkdirAll(filepath.Dir(config), 0644)
	if err != nil {
		return err
	}
	f := filepath.Base(config)
	conf := fmt.Sprintf("/configmaps/%s/%s", f, f)
	t := template.Must(template.New(f).ParseFiles(conf))

	nconf, err := os.Create(config)
	if err != nil {
		return err
	}
	params := make(map[string]string)
	iface, err := EnvExists("INTERFACE_NAME")
	if err != nil {
		return err
	}
	ip, err := GetIpFromInterface(iface)
	params["IP"] = ip
	params["IP_ERLANG"] = strings.Replace(ip, ".", ",", -1)
	params["HOSTNAME"] = os.Getenv("HOSTNAME")

	if err != nil {
		return err
	}
	err = t.Execute(nconf, params)
	if err != nil {
		return err
	}
	return nil
}
func EnvExists(env string) (string, error) {
	e := os.Getenv(env)
	if e == "" {
		return "", fmt.Errorf("Environment variable %s is empty", env)
	}
	return e, nil
}

func WaitForService(c *client.Client, namespace string, deps []string) error {

	seviceDepState := WAITING
	if deps == nil {
		seviceDepState = READY
		Info.Println("Container has no service dependencies")
	}

	for seviceDepState == WAITING {

		for i := range deps {

			service := strings.TrimSpace(deps[i])
			err := CheckIfServiceExists(c, namespace, service)
			if err != nil {
				Info.Println(service, " doesn't exists -> State waiting")
				seviceDepState = WAITING
				break
			}
			a, err := CheckEndpointsAvailabilty(c, namespace, service)
			if err != nil {
				Info.Println(service, " doesn't exist -> State waiting")
				seviceDepState = WAITING
				break
			}
			if !a {
				Info.Println(service, " service has no endpoints avaiable -> State waiting")
				seviceDepState = WAITING
				break
			}

			Info.Println(service, " service has at least one running endpoint. -> State ready")
			seviceDepState = READY

		}

	}
	Info.Println("All dependencies resolved")
	return nil

}

func RenderConfigs(configs []string) error {
	if configs == nil {
		Info.Println("Container has no configs to render")
	}
	for i := range configs {
		err := RenderConfigWithIP(configs[i])
		if err != nil {
			return err
		}
		Info.Println("Rendering config: ", configs[i])
	}
	return nil
}

func WaitForJobs(c *client.Client, namespace string, jobs []string) error {
	jobState := WAITING
	if jobs == nil {
		jobState = READY
		Info.Println("Container has no jobs dependencies")
	}

	for jobState == WAITING {

		for _, job := range jobs {

			j := strings.TrimSpace(job)
			err := CheckIfJobExists(c, namespace, j)
			if err != nil {
				return err
			}
			a, err := IsJobComplete(c, namespace, j)
			if err != nil {
				Info.Println(j, " doesn't exists -> State waiting")
				jobState = WAITING
				break
			}
			if !a {
				Info.Println(j, " job is not complete -> State waiting")
				jobState = WAITING
				break
			}

			Info.Println(j, " job is completed -> State ready")
			jobState = READY

		}

	}
	Info.Println("All jobs dependencies resolved")
	return nil

}

func InitLogger(linfo io.Writer, lerror io.Writer) {
	Info = log.New(linfo, "Entrypoint INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(lerror, "Entrypoint Error: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {

	//Set Logger
	InitLogger(os.Stdout, os.Stderr)
	//Those envs should be set as DownwardAPI http://kubernetes.io/docs/user-guide/downward-api/
	// podName, err := EnvExists("POD_NAME")
	// if err != nil {
	// 	Error.Println(err)
	// }
	namespace, err := EnvExists("NAMESPACE")
	if err != nil {
		Error.Println(err)
	}

	// Inside k8s POD we need to initialise client with such function
	c, err := client.NewInCluster()
	// For testing purposes uncomment following section and comment out above and fill Host property
	// config := &restclient.Config{
	// 	Host: "http://127.0.0.1:8080",
	// }
	// c, err := client.New(config)

	// if err != nil {
	// 	Error.Println(err)
	// 	os.Exit(1)
	// }

	// p, err := c.Pods(namespace).Get(podName)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}
	// jobs := GetAnnotations(p.Annotations, "jobs_dependencies")
	jobs := GetEnv("JOBS")
	err = WaitForJobs(c, namespace, jobs)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}
	// serviceDeps := GetAnnotations(p.Annotations, "service_dependencies")
	serviceDeps := GetEnv("SERVICES")
	err = WaitForService(c, namespace, serviceDeps)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}
	// configs := GetAnnotations(p.Annotations, "configs")
	configs := GetEnv("CONFIGS")
	err = RenderConfigs(configs)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}

	command := GetEnv("COMMAND", " ")
	err = ExecuteCommand(command)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}

}
