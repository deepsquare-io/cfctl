package phase

import (
	"github.com/SquareFactory/cfctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"

	// anonymous import is needed to load the os configurers
	_ "github.com/SquareFactory/cfctl/configurer"
	// anonymous import is needed to load the os configurers
	_ "github.com/SquareFactory/cfctl/configurer/linux"
	// anonymous import is needed to load the os configurers
	_ "github.com/SquareFactory/cfctl/configurer/linux/enterpriselinux"

	log "github.com/sirupsen/logrus"
)

// DetectOS performs remote OS detection
type DetectOS struct {
	GenericPhase
}

// Title for the phase
func (p *DetectOS) Title() string {
	return "Detect host operating systems"
}

// Run the phase
func (p *DetectOS) Run() error {
	return p.Config.Spec.Hosts.ParallelEach(func(h *cluster.Host) error {
		if h.OSIDOverride != "" {
			log.Infof("%s: OS ID has been manually set to %s", h, h.OSIDOverride)
		}
		if err := h.ResolveConfigurer(); err != nil {
			p.SetProp("missing-support", h.OSVersion.String())
			return err
		}
		os := h.OSVersion.String()
		p.IncProp(os)
		log.Infof("%s: is running %s", h, os)

		return nil
	})
}
