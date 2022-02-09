package config

import (
	"bytes"
	"text/template"

	v1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

const configFormat = `
SocksPort 0
HiddenServiceDir {{ .ServiceDir }}
{{ if .HiddenServiceOnionbalanceInstance }}
HiddenServiceOnionbalanceInstance 1
{{ end }}
HiddenServiceVersion {{ .Version }}
{{ range .Ports }}
HiddenServicePort {{ .PublicPort }} {{ .ServiceClusterIP }}:{{ .ServicePort }}
{{ end }}
`

const oBconfigFormat = `
{{ if .HiddenServiceOnionbalanceInstance }}
MasterOnionAddress {{.MasterOnionAddress}}
{{ end }}
`

var configTemplate = template.Must(template.New("config").Parse(configFormat))
var oBconfigTemplate = template.Must(template.New("config").Parse(oBconfigFormat))

type onionService struct {
	ServiceName                       string
	ServiceNamespace                  string
	ServiceDir                        string
	Version                           int
	Ports                             []portTuple
	MasterOnionAddress                string
	HiddenServiceOnionbalanceInstance bool
}

type portTuple struct {
	ServicePort      int32
	PublicPort       int32
	ServiceClusterIP string
}

func OnionServiceInputData(onion *v1alpha2.OnionService) onionService {
	ports := []portTuple{}
	for _, rule := range onion.Spec.Rules {
		port := portTuple{
			ServicePort:      rule.Backend.Service.Port.Number,
			PublicPort:       rule.Port.Number,
			ServiceClusterIP: rule.Backend.Service.Name,
		}
		ports = append(ports, port)
	}

	return onionService{
		ServiceName:                       onion.ServiceName(),
		ServiceNamespace:                  onion.Namespace,
		ServiceDir:                        "/run/tor/service",
		Ports:                             ports,
		Version:                           onion.Spec.GetVersion(),
		MasterOnionAddress:                onion.Spec.MasterOnionAddress,
		HiddenServiceOnionbalanceInstance: onion.Spec.MasterOnionAddress != "",
	}
}

func TorConfigForService(onion *v1alpha2.OnionService) (string, error) {
	s := OnionServiceInputData(onion)
	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, s)
	if err != nil {
		return "", err
	}
	return tmp.String(), nil
}

func ObConfigForService(onion *v1alpha2.OnionService) (string, error) {
	s := OnionServiceInputData(onion)
	var tmp bytes.Buffer
	err := oBconfigTemplate.Execute(&tmp, s)
	if err != nil {
		return "", err
	}
	return tmp.String(), nil
}
