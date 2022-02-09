package config

import (
	"fmt"

	v1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
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
		if len(b.Hostname) != 0 {
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
		fmt.Printf("Error while Marshaling. %v", err)
	}

	if err != nil {
		return "", err
	}
	return string(yamlData), nil
}
