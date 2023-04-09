package config

import (
	log "github.com/sirupsen/logrus"

	v1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Services []Service `yaml:"services"`
}

type Service struct {
	Instances []Instance `yaml:"instances"`
	Key       string     `yaml:"key"`
}

type Instance struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

func OnionBalanceConfigForService(onion *v1alpha2.OnionBalancedService) (string, error) {
	instances := []Instance{}

	for name, b := range onion.Status.Backends {
		if b.Hostname != "" {
			instances = append(instances, Instance{Name: name, Address: b.Hostname})
		}
	}

	config := Config{
		Services: []Service{
			{
				Instances: instances,
				Key:       "key/privateKeyFile",
			},
		},
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		log.Printf("Error while Marshaling. %v", err)

		return "", errors.Wrap(err, "Error while Marshaling. %v")
	}

	return string(yamlData), nil
}
