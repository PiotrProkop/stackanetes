package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	//For testing purposes
	restclient "k8s.io/kubernetes/pkg/client/restclient"
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
func ExecuteCommandFromAnnotation(command []string) error {
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

	cmd.Run()
	return nil
}

//This function retrives a command section and dependencies section from k8s annotations
func GetAnnotations(annotations map[string]string) (command []string, serviceDeps []string, configs []string) {
	command = strings.Split(annotations["command"], " ")
	if len(command) == 0 || command[0] == "" {
		Error.Println("Command not specified")
		os.Exit(1)
	}
	serviceDeps = strings.Split(annotations["service_dependencies"], ",")
	if len(serviceDeps) == 0 || serviceDeps[0] == "" {
		serviceDeps = nil
	}
	configs = strings.Split(annotations["configs"], ",")
	if len(configs) == 0 || configs[0] == "" {
		configs = nil
	}
	return command, serviceDeps, configs
}

//This function check if a service in given namespace exists
func CheckIfServiceExists(c *client.Client, namespace string, service string) {

	_, err := c.Services(namespace).Get(service)
	if err != nil {
		Error.Println(service, "service doesn't exist in", namespace, "namespace. Error:", err)
		os.Exit(1)
	}
}

//This function check if given service has at least one endpoint active
func CheckEndpointsAvailabilty(c *client.Client, namespace string, service string) bool {

	e, err := c.Endpoints(namespace).Get(service)
	if err != nil {
		Error.Println(service, "service doesn't exist in", namespace, "namespace. Error: ", err)
		os.Exit(1)
	}
	if len(e.Subsets) == 0 {
		return false
	}
	return true
}

//"GetIpFromInterface return always first ip from interface"
func GetIpFromInterface(iface string) string {

	i, err := net.InterfaceByName(iface)
	if err != nil {
		Error.Println(iface, "interface doesn't exist. Error: ", err)
		os.Exit(1)
	}
	addr, err := i.Addrs()
	if err != nil || len(addr) == 0 {
		Error.Println(iface, " interface doesn't have ip set. Error: ", err)
		os.Exit(1)
	}
	return strings.Split(addr[0].String(), "/")[0]
}

func RenderConfigWithIP(config string) {

	t := template.Must(template.New(filepath.Base(config)).ParseFiles(config))
	file, err := os.OpenFile(config, os.O_RDWR, os.ModeCharDevice)
	if err != nil {
		Error.Println(err)
	}
	params := make(map[string]string)
	params["IP"] = GetIpFromInterface(EnvExists("INTERFACE_NAME"))
	err = t.Execute(file, params)
	if err != nil {
		Error.Println(err)
	}

}
func EnvExists(env string) string {
	e := os.Getenv(env)
	if e == "" {
		Error.Println("Environment variable ", env, " is empty")
		os.Exit(1)
	}
	return e
}

func WaitForServiceDependency(c *client.Client, namespace string, deps []string) {

	seviceDepState := WAITING
	if deps == nil {
		seviceDepState = READY
		Info.Println("Container has no service dependencies")
	}

	for seviceDepState == WAITING {

		for i := range deps {

			service := strings.TrimSpace(deps[i])
			CheckIfServiceExists(c, namespace, service)

			if !CheckEndpointsAvailabilty(c, namespace, service) {
				Info.Println(service, " service has no endpoints avaiable -> State waiting")
				seviceDepState = WAITING
				break
			}

			Info.Println(service, " service has at least one running endpoint. -> State ready")
			seviceDepState = READY

		}
		Info.Println("All dependencies resolved")
	}

}

func RenderConfigs(configs []string) {
	if configs == nil {
		Info.Println("Container has no configs to render")
	}
	for i := range configs {
		RenderConfigWithIP(configs[i])
		Info.Println("Rendering config: ", configs[i])
	}
}

func InitLogger(linfo io.Writer, lerror io.Writer) {
	Info = log.New(linfo, "Entrypoint INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(lerror, "Entrypoint Error: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	//Those envs should be set as DownwardAPI http://kubernetes.io/docs/user-guide/downward-api/
	podName := EnvExists("POD_NAME")
	namespace := EnvExists("NAMESPACE")

	//Set Logger
	InitLogger(os.Stdout, os.Stderr)
	// Inside k8s POD we need to initialise client with such function
	c, err := client.NewInCluster()
	// For testing purposes uncomment following section and comment out above and fill Host property
	// config := &restclient.Config{
	// 	Host: "http://127.0.0.1:8080",
	// }
	// c, err := client.New(config)

	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}

	p, err := c.Pods(namespace).Get(podName)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}

	// renderConfigWithIP("/home/pprokop/Templates/test.conf")
	command, serviceDeps, configs := GetAnnotations(p.Annotations)
	WaitForServiceDependency(c, namespace, serviceDeps)
	RenderConfigs(configs)
	err = ExecuteCommandFromAnnotation(command)
	if err != nil {
		Error.Println(err)
		os.Exit(1)
	}

}
