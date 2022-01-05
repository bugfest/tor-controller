package config

import (
	"bytes"
	"text/template"

	v1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

const configFormat = `
SocksPort 0
HiddenServiceDir {{ .ServiceDir }}
HiddenServiceVersion {{ .Version }}
{{ range .Ports }}
HiddenServicePort {{ .PublicPort }} {{ .ServiceClusterIP }}:{{ .ServicePort }}
{{ end }}
`

var configTemplate = template.Must(template.New("config").Parse(configFormat))

type onionService struct {
	ServiceName      string
	ServiceNamespace string
	ServiceDir       string
	Version          int
	Ports            []portTuple
}

type portTuple struct {
	ServicePort      int32
	PublicPort       int32
	ServiceClusterIP string
}

func TorConfigForService(onion *v1alpha2.OnionService) (string, error) {
	ports := []portTuple{}
	for _, rule := range onion.Spec.Rules {
		port := portTuple{
			ServicePort:      rule.Backend.Service.Port.Number,
			PublicPort:       rule.Port.Number,
			ServiceClusterIP: rule.Backend.Service.Name,
		}
		ports = append(ports, port)
	}

	s := onionService{
		ServiceName:      onion.ServiceName(),
		ServiceNamespace: onion.Namespace,
		ServiceDir:       "/run/tor/service",
		Ports:            ports,
		Version:          onion.Spec.GetVersion(),
	}

	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, s)
	if err != nil {
		return "", err
	}
	return tmp.String(), nil
}
