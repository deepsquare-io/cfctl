package phase

import (
	"context"
	"fmt"
	"time"

	"github.com/deepsquare-io/cfctl/pkg/apis/cfctl.clusterfactory.io/v1beta1"
	"github.com/deepsquare-io/cfctl/pkg/apis/cfctl.clusterfactory.io/v1beta1/cluster"
	"github.com/deepsquare-io/cfctl/pkg/node"
	"github.com/deepsquare-io/cfctl/pkg/retry"
	log "github.com/sirupsen/logrus"
)

// UpgradeControllers upgrades the controllers one-by-one
type UpgradeControllers struct {
	GenericPhase

	hosts cluster.Hosts
}

// Title for the phase
func (p *UpgradeControllers) Title() string {
	return "Upgrade controllers"
}

// Prepare the phase
func (p *UpgradeControllers) Prepare(config *v1beta1.Cluster) error {
	log.Debugf("UpgradeControllers phase prep starting")
	p.Config = config
	var controllers cluster.Hosts = p.Config.Spec.Hosts.Controllers()
	log.Debugf("%d controllers in total", len(controllers))
	p.hosts = controllers.Filter(func(h *cluster.Host) bool {
		if h.Metadata.K0sBinaryTempFile == "" {
			return false
		}
		return !h.Reset && h.Metadata.NeedsUpgrade
	})
	log.Debugf("UpgradeControllers phase prepared, %d controllers needs upgrade", len(p.hosts))
	return nil
}

// ShouldRun is true when there are controllers that needs to be upgraded
func (p *UpgradeControllers) ShouldRun() bool {
	return len(p.hosts) > 0
}

// CleanUp cleans up the environment override files on hosts
func (p *UpgradeControllers) CleanUp() {
	for _, h := range p.hosts {
		if len(h.Environment) > 0 {
			if err := h.Configurer.CleanupServiceEnvironment(h, h.K0sServiceName()); err != nil {
				log.Warnf("%s: failed to clean up service environment: %s", h, err.Error())
			}
		}
	}
}

// Run the phase
func (p *UpgradeControllers) Run() error {
	for _, h := range p.hosts {
		if !h.Configurer.FileExist(h, h.Metadata.K0sBinaryTempFile) {
			return fmt.Errorf("k0s binary tempfile not found on host")
		}
		log.Infof("%s: starting upgrade", h)
		log.Debugf("%s: stop service", h)
		err := p.Wet(h, "stop k0s service", func() error {
			if err := h.Configurer.StopService(h, h.K0sServiceName()); err != nil {
				return err
			}
			if err := retry.Timeout(context.TODO(), retry.DefaultTimeout, node.ServiceStoppedFunc(h, h.K0sServiceName())); err != nil {
				return fmt.Errorf("wait for k0s service stop: %w", err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		log.Debugf("%s: update binary", h)
		err = p.Wet(h, "replace k0s binary", func() error {
			return h.UpdateK0sBinary(h.Metadata.K0sBinaryTempFile, p.Config.Spec.K0s.Version)
		})
		if err != nil {
			return err
		}

		if len(h.Environment) > 0 {
			log.Infof("%s: updating service environment", h)
			err := p.Wet(h, "update service environment", func() error {
				return h.Configurer.UpdateServiceEnvironment(h, h.K0sServiceName(), h.Environment)
			})
			if err != nil {
				return err
			}
		}

		log.Debugf("%s: restart service", h)
		err = p.Wet(h, "start k0s service with the new binary", func() error {
			if err := h.Configurer.StartService(h, h.K0sServiceName()); err != nil {
				return err
			}
			log.Infof("%s: waiting for the k0s service to start", h)
			if err := retry.Timeout(context.TODO(), retry.DefaultTimeout, node.ServiceRunningFunc(h, h.K0sServiceName())); err != nil {
				return fmt.Errorf("k0s service start: %w", err)
			}
			return nil
		})
		if err != nil {
			return err
		}
		port := 6443
		if p, ok := p.Config.Spec.K0s.Config.Dig("spec", "api", "port").(int); ok {
			port = p
		}

		if p.IsWet() {
			if err := retry.Timeout(context.TODO(), retry.DefaultTimeout, node.KubeAPIReadyFunc(h, port)); err != nil {
				return fmt.Errorf("kube api did not become ready: %w", err)
			}
		}
	}

	leader := p.Config.Spec.K0sLeader()
	if NoWait || !p.IsWet() {
		log.Warnf("%s: skipping scheduler and system pod checks because --no-wait given", leader)
		return nil
	}

	log.Infof("%s: waiting for the scheduler to become ready", leader)
	if err := retry.Timeout(context.TODO(), retry.DefaultTimeout, node.ScheduledEventsAfterFunc(leader, time.Now())); err != nil {
		if !Force {
			return fmt.Errorf(
				"failed to observe scheduling events after api start-up, you can ignore this check by using --force: %w",
				err,
			)
		}
		log.Warnf("%s: failed to observe scheduling events after api start-up: %s", leader, err)
	}

	log.Infof("%s: waiting for system pods to become ready", leader)
	if err := retry.Timeout(context.TODO(), retry.DefaultTimeout, node.SystemPodsRunningFunc(leader)); err != nil {
		if !Force {
			return fmt.Errorf(
				"all system pods not running after api start-up, you can ignore this check by using --force: %w",
				err,
			)
		}
		log.Warnf("%s: failed to observe system pods running after api start-up: %s", leader, err)
	}

	return nil
}
