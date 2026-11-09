package nsfContext

import (
	"encoding/json"
	"fmt"
	"nsf/internal/db"
	"nsf/logger"
)

// TODO: You may need to add some fields
type serviceContextIE struct {
	db.DbIf

	*logger.NsfLogger
}

// TODO: You may need to add some fields
type serviceContext struct {
	db.DbIf

	*logger.NsfLogger
}

type serviceInfo struct {
	NfName     string `json:"nf_name"`
	SbiAddress string `json:"sbi_address"`
	SbiPort    int    `json:"sbi_port"`
}

// TODO: Finish the initialization of the serviceContext instance
func newServiceContext(serviceContextIE *serviceContextIE) *serviceContext {

	return &serviceContext{
		DbIf: serviceContextIE.DbIf,

		NsfLogger: serviceContextIE.NsfLogger,
	}
}

// TODO: Finish the release of the serviceContext instance
func (s *serviceContext) release() {
	s.NsfLogger.SvcLog.Infoln("Release serviceContext...")

	s.NsfLogger.SvcLog.Infoln("serviceContext released")
}

// TODO: Finish the implementation of the registerServices method with an error return type
func (s *serviceContext) registerServices(nfName, sbiAddr string, sbiPort int, services []string) error {
	s.NsfLogger.SvcLog.Tracef("Registering %s services", nfName)

	data := make(map[string]string, len(services))
	for _, service := range services {
		info := serviceInfo{
			NfName: nfName, SbiAddress: sbiAddr, SbiPort: sbiPort,
		}
		b, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", service, err)
		}
		data[service] = string(b)
	}
	return s.DbIf.SaveAll(data)
}

// TODO: Finish the implementation of the deregisterServices method with an error return type
func (s *serviceContext) deregisterServices(nfName string, services []string) error {
	s.NsfLogger.SvcLog.Tracef("De-registering %s services", nfName)

	for _, service := range services {
		if err := s.DbIf.Remove(service); err != nil {
			return fmt.Errorf("remove service %s: %w", service, err)
		}
	}
	return nil
}

// TODO: Finish the implementation of the discoverService method with these return values:
// (string, string, int, error) representing the NF name, SBI address, SBI port, and an error if any
func (s *serviceContext) discoverService(service string) (string, string, int, error) {
	s.NsfLogger.SvcLog.Tracef("Discovering service %s", service)

	exist, err := s.DbIf.Exist(service)
	if err != nil {
		return "", "", 0, fmt.Errorf("check service %s: %w", service, err)
	}
	if !exist {
		return "", "", 0, fmt.Errorf("service %s not found", service)
	}

	data, err := s.DbIf.Load(service)
	if err != nil {
		return "", "", 0, fmt.Errorf("load service %s: %w", service, err)
	}

	var info serviceInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return "", "", 0, fmt.Errorf("unmarshal service %s: %w", service, err)
	}
	return info.NfName, info.SbiAddress, info.SbiPort, nil
}
