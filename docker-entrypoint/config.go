package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type config struct {
	params struct {
		iface     string
		HOSTNAME  string
		IP        string
		IP_ERLANG string
	}
	configs []string
}

func NewConfig() (conf *config, err error) {
	configs := ConvertEnvToList("CONFIGS")
	iface := os.Getenv("INTERFACE_NAME")

	if iface == "" {
		return nil, fmt.Errorf("Environment variable INTERFACE_NAME is empty")
	}
	conf = new(config)
	conf.params.iface = iface
	if configs == nil {
		return conf, nil
	}
	err = conf.SetIps()
	if err != nil {
		return nil, err
	}
	conf.configs = configs
	conf.params.HOSTNAME = os.Getenv("HOSTNAME")
	return conf, nil
}

func (conf *config) SetIps() error {

	i, err := net.InterfaceByName(conf.params.iface)
	if err != nil {
		return err
	}
	addr, err := i.Addrs()
	if err != nil || len(addr) == 0 {
		return err
	}
	conf.params.IP = strings.Split(addr[0].String(), "/")[0]
	conf.params.IP_ERLANG = strings.Replace(conf.params.IP, ".", ",", -1)
	return nil
}

func CreateDirectory(config string) error {
	err := os.MkdirAll(filepath.Dir(config), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (conf config) RenderConfigs() error {
	if conf.configs == nil {
		return nil
	}
	for _, con := range conf.configs {
		Info.Println("Rendering config: ", con)
		err := CreateDirectory(con)
		if err != nil {
			return err
		}
		out, err := os.Create(con)
		if err != nil {
			return err
		}
		f := filepath.Base(con)
		t := template.Must(template.New(f).ParseFiles(fmt.Sprintf("/configmaps/%s/%s", f, f)))
		err = t.Execute(out, conf.params)
		if err != nil {
			return err
		}
	}
	return nil
}
