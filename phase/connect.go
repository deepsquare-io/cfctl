package phase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/deepsquare-io/cfctl/pkg/apis/cfctl.clusterfactory.io/v1beta1/cluster"
	"github.com/deepsquare-io/cfctl/pkg/retry"
	"github.com/k0sproject/rig"
	log "github.com/sirupsen/logrus"
)

// Connect connects to each of the hosts
type Connect struct {
	GenericPhase
}

// Title for the phase
func (p *Connect) Title() string {
	return "Connect to hosts"
}

// Run the phase
func (p *Connect) Run() error {
	return p.parallelDo(p.Config.Spec.Hosts, func(h *cluster.Host) error {
		return retry.Timeout(context.TODO(), 10*time.Minute, func(_ context.Context) error {
			if err := h.Connect(); err != nil {
				if errors.Is(err, rig.ErrCantConnect) ||
					strings.Contains(err.Error(), "host key mismatch") {
					return errors.Join(retry.ErrAbort, err)
				}

				return err
			}

			log.Infof("%s: connected", h)

			return nil
		})
	})
}
