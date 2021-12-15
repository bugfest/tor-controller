package v1alpha2

import "fmt"

const (
	deploymentNameFmt     = "%s-tor-daemon"
	secretNameFmt         = "%s-tor-secret"
	serviceNameFmt        = "%s-tor-svc"
	roleNameFmt           = "%s-tor-role"
	serviceAccountNameFmt = "%s-tor-serviceaccount"
)

func (s *OnionServiceSpec) GetVersion() int {
	v := 3
	if s.Version == 2 {
		v = 2
	}
	return v
}

func (s *OnionService) DeploymentName() string {
	return fmt.Sprintf(deploymentNameFmt, s.Name)
}

func (s *OnionService) ServiceName() string {
	return fmt.Sprintf(serviceNameFmt, s.Name)
}

func (s *OnionService) SecretName() string {
	if len(s.Spec.PrivateKeySecret.Name) > 0 {
		return s.Spec.PrivateKeySecret.Name
	}
	return fmt.Sprintf(secretNameFmt, s.Name)
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
	return fmt.Sprintf(roleNameFmt, s.Name)
}

func (s *OnionService) ServiceAccountName() string {
	return fmt.Sprintf(serviceAccountNameFmt, s.Name)
}
