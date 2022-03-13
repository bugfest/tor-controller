package v1alpha2

import "fmt"

const (
	osDeploymentNameFmt     = "%s-tor-daemon"
	osSecretNameFmt         = "%s-tor-secret"
	osServiceNameFmt        = "%s-tor-svc"
	osMetricsServiceNameFmt = "%s-tor-metrics-svc"
	osRoleNameFmt           = "%s-tor-role"
	osServiceAccountNameFmt = "%s-tor-sa"
	osServiceBackendNameFmt = "%s-tor-obb-%d"
)

func (s *OnionServiceSpec) GetVersion() int {
	v := 3
	if s.Version == 2 {
		v = 2
	}
	return v
}

func (s *OnionBalancedService) OnionServiceBackendName(n int32) string {
	return fmt.Sprintf(osServiceBackendNameFmt, s.Name, n)
}

func (s *OnionService) DeploymentName() string {
	return fmt.Sprintf(osDeploymentNameFmt, s.Name)
}

func (s *OnionService) ServiceName() string {
	return fmt.Sprintf(osServiceNameFmt, s.Name)
}

func (s *OnionService) ServiceMetricsName() string {
	return fmt.Sprintf(osMetricsServiceNameFmt, s.Name)
}

func (s *OnionService) ServiceMetricsSelector() map[string]string {
	return map[string]string{
		"app":        s.ServiceMetricsName(),
		"controller": s.Name,
	}
}

func (s *OnionService) SecretName() string {
	if len(s.Spec.PrivateKeySecret.Name) > 0 {
		return s.Spec.PrivateKeySecret.Name
	}
	return fmt.Sprintf(osSecretNameFmt, s.Name)
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
	return fmt.Sprintf(osRoleNameFmt, s.Name)
}

func (s *OnionService) ServiceAccountName() string {
	return fmt.Sprintf(osServiceAccountNameFmt, s.Name)
}
