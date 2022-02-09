package v1alpha2

import "fmt"

const (
	torDeploymentNameFmt               = "%s-tor-daemon"
	torSecretNameFmt                   = "%s-tor-secret"
	torServiceNameFmt                  = "%s-tor-svc"
	torRoleNameFmt                     = "%s-tor-role"
	torServiceAccountNameFmt           = "%s-tor-serviceaccount"
	onionBalancedServiceBackendNameFmt = "%s-obb-%d"
)

func (s *OnionServiceSpec) GetVersion() int {
	v := 3
	if s.Version == 2 {
		v = 2
	}
	return v
}

func (s *OnionBalancedService) OnionServiceBackendName(n int32) string {
	return fmt.Sprintf(onionBalancedServiceBackendNameFmt, s.Name, n)
}

func (s *OnionService) DeploymentName() string {
	return fmt.Sprintf(torDeploymentNameFmt, s.Name)
}

func (s *OnionService) ServiceName() string {
	return fmt.Sprintf(torServiceNameFmt, s.Name)
}

func (s *OnionService) SecretName() string {
	if len(s.Spec.PrivateKeySecret.Name) > 0 {
		return s.Spec.PrivateKeySecret.Name
	}
	return fmt.Sprintf(torSecretNameFmt, s.Name)
}

func (s *OnionService) ServiceSelector() map[string]string {
	serviceSelector := map[string]string{
		"app":        s.ServiceName(),
		"controller": s.Name,
	}
	return serviceSelector
}

func (s *OnionService) DeploymentLabels() map[string]string {
	return s.ServiceSelector()
}

func (s *OnionService) RoleName() string {
	return fmt.Sprintf(torRoleNameFmt, s.Name)
}

func (s *OnionService) ServiceAccountName() string {
	return fmt.Sprintf(torServiceAccountNameFmt, s.Name)
}
