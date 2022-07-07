package config

import (
	"bytes"
	"text/template"

	v1alpha3 "github.com/bugfest/tor-controller/apis/tor/v1alpha3"
)

const configFormat = `
SocksPort {{ .SocksPort }}
ControlPort {{ .ControlPort }}
MetricsPort {{ .MetricsPort }}
MetricsPortPolicy {{ .MetricsPortPolicy }}
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

type torConfig struct {
	SocksPort                         string
	ControlPort                       string
	MetricsPort                       string
	MetricsPortPolicy                 string
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

func OnionServiceInputData(onion *v1alpha3.OnionService) torConfig {
	ports := []portTuple{}
	for _, rule := range onion.Spec.Rules {
		port := portTuple{
			ServicePort:      rule.Backend.Service.Port.Number,
			PublicPort:       rule.Port.Number,
			ServiceClusterIP: rule.Backend.Service.Name,
		}
		ports = append(ports, port)
	}

	return torConfig{
		SocksPort:                         "0",
		ControlPort:                       "0",
		MetricsPort:                       "0.0.0.0:9035",
		MetricsPortPolicy:                 "accept 0.0.0.0/0",
		ServiceName:                       onion.ServiceName(),
		ServiceNamespace:                  onion.Namespace,
		ServiceDir:                        "/run/tor/service",
		Ports:                             ports,
		Version:                           onion.Spec.GetVersion(),
		MasterOnionAddress:                onion.Spec.MasterOnionAddress,
		HiddenServiceOnionbalanceInstance: onion.Spec.MasterOnionAddress != "",
	}
}

func TorConfigForService(onion *v1alpha3.OnionService) (string, error) {
	s := OnionServiceInputData(onion)
	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, s)
	if err != nil {
		return "", err
	}
	return tmp.String(), nil
}

// Generates ob_config file if this instance handles traffic on behalf of a master hidden service
func ObConfigForService(onion *v1alpha3.OnionService) (string, error) {
	s := OnionServiceInputData(onion)
	var tmp bytes.Buffer
	err := oBconfigTemplate.Execute(&tmp, s)
	if err != nil {
		return "", err
	}
	return tmp.String(), nil
}
