package v1alpha2

import "fmt"

const (
	onionbalanceDeploymentNameFmt     = "%s-tor-daemon"
	onionbalanceSecretNameFmt         = "%s-tor-secret"
	onionbalanceServiceNameFmt        = "%s-tor-svc"
	onionbalanceRoleNameFmt           = "%s-tor-role"
	onionbalanceServiceAccountNameFmt = "%s-tor-serviceaccount"
	onionbalanceConfigMapFmt          = "%s-tor-config"
)

func (s *OnionBalancedServiceSpec) GetVersion() int {
	v := 3
	if s.Version == 2 {
		v = 2
	}
	return v
}

func (s *OnionBalancedServiceSpec) GetReplicas() int {
	return int(s.Replicas)
}

func (s *OnionBalancedService) DeploymentName() string {
	return fmt.Sprintf(torDeploymentNameFmt, s.Name)
}

func (s *OnionBalancedService) ConfigMapName() string {
	return fmt.Sprintf(onionbalanceConfigMapFmt, s.Name)
}

func (s *OnionBalancedService) ServiceName() string {
	return fmt.Sprintf(torServiceNameFmt, s.Name)
}

func (s *OnionBalancedService) SecretName() string {
	if s.Spec.PrivateKeySecret != (SecretReference{}) {
		if len(s.Spec.PrivateKeySecret.Name) > 0 {
			return s.Spec.PrivateKeySecret.Name
		}
	}
	return fmt.Sprintf(torSecretNameFmt, s.Name)
}

func (s *OnionBalancedService) ServiceSelector() map[string]string {
	serviceSelector := map[string]string{
		"app":        s.ServiceName(),
		"controller": s.Name,
	}
	return serviceSelector
}

func (s *OnionBalancedService) DeploymentLabels() map[string]string {
	return s.ServiceSelector()
}

func (s *OnionBalancedService) RoleName() string {
	return fmt.Sprintf(torRoleNameFmt, s.Name)
}

func (s *OnionBalancedService) ServiceAccountName() string {
	return fmt.Sprintf(torServiceAccountNameFmt, s.Name)
}

func (s *OnionBalancedService) IsSynced() bool {
	// All backends must exist
	if len(s.Status.Backends) != s.Spec.GetReplicas() {
		return false
	}
	// All backends must have a hostname
	for _, backend := range s.Status.Backends {
		if len(backend.Hostname) == 0 {
			return false
		}
	}
	return true
}
