package cmd

import "net"

// ChangeGlobalIP represents the changed state of the global IP
type ChangeGlobalIP struct {
	PreviousGlobalIPv4 string `json:"previousGlobalIPv4,omitempty"`
	CurrentGlobalIPv4  string `json:"currentGlobalIPv4,omitempty"`
}

// NewChangeGlobalIP creates a ChangeGlobalIP structure
func NewChangeGlobalIP(pIP net.IP, cIP net.IP) *ChangeGlobalIP {
	var (
		ps string
		cs string
	)
	if pIP != nil {
		ps = pIP.String()
	}
	if cIP != nil {
		cs = cIP.String()
	}

	return &ChangeGlobalIP{
		PreviousGlobalIPv4: ps,
		CurrentGlobalIPv4:  cs,
	}
}
