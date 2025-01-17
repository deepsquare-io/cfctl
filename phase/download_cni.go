package phase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/deepsquare-io/cfctl/pkg/apis/cfctl.clusterfactory.io/v1beta1"
	"github.com/deepsquare-io/cfctl/pkg/apis/cfctl.clusterfactory.io/v1beta1/cluster"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

var _ phase = &DownloadCNI{}

type DownloadCNI struct {
	GenericPhase
	hosts []*cluster.Host
}

// Title returns the title for the phase
func (p *DownloadCNI) Title() string {
	return "Download the CNIs"
}

// Prepare the phase
func (p *DownloadCNI) Prepare(config *v1beta1.Cluster) error {
	p.Config = config
	p.hosts = p.Config.Spec.Hosts
	return nil
}

func (p *DownloadCNI) ensureDir(h *cluster.Host, dir, perm, owner string) error {
	log.Debugf("%s: ensuring directory %s", h, dir)
	if h.Configurer.FileExist(h, dir) {
		return nil
	}

	err := p.Wet(
		h,
		fmt.Sprintf("create a directory for uploading: `mkdir -p \"%s\"`", dir),
		func() error {
			return h.Configurer.MkDir(h, dir, exec.Sudo(h))
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if perm == "" {
		perm = "0755"
	}

	err = p.Wet(h, fmt.Sprintf("set permissions for directory %s to %s", dir, perm), func() error {
		return h.Configurer.Chmod(h, dir, perm, exec.Sudo(h))
	})
	if err != nil {
		return fmt.Errorf("failed to set permissions for directory %s: %w", dir, err)
	}

	if owner != "" {
		err = p.Wet(h, fmt.Sprintf("set owner for directory %s to %s", dir, owner), func() error {
			return h.Execf(`chown "%s" "%s"`, owner, dir, exec.Sudo(h))
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Run the phase
func (p *DownloadCNI) Run() error {
	for _, h := range p.hosts {
		if err := p.ensureDir(h, "/opt/cni/bin", "0755", "0"); err != nil {
			return err
		}

		b, err := fetchLatestCNIVersion(h)
		if err != nil {
			return err
		}

		if err := h.Exec(fmt.Sprintf(`curl -sSLf %s | tar -C /opt/cni/bin/ -xzf -`, shellescape.Quote(b.url())), exec.Sudo(h)); err != nil {
			return err
		}
	}
	return nil
}

type cniBinary struct {
	Arch    string `json:"-"`
	OS      string `json:"-"`
	Version string `json:"tag_name"`
}

func fetchLatestCNIVersion(h *cluster.Host) (cniBinary, error) {
	resp, err := http.Get(
		"https://api.github.com/repos/containernetworking/plugins/releases/latest",
	)
	if err != nil {
		return cniBinary{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cniBinary{}, fmt.Errorf(
			"failed to get latest CNI plugins version (http %d)",
			resp.StatusCode,
		)
	}

	var result = cniBinary{
		Arch: h.Metadata.Arch,
		OS:   h.Configurer.Kind(),
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return cniBinary{}, errors.New(
			"failed to get latest CNI plugins version (couldn't decode response)",
		)
	}

	return result, nil
}

func (b cniBinary) url() string {
	return fmt.Sprintf(
		"https://github.com/containernetworking/plugins/releases/download/v%s/cni-plugins-%s-%s-v%s.tgz",
		strings.TrimPrefix(b.Version, "v"),
		b.OS,
		b.Arch,
		strings.TrimPrefix(b.Version, "v"),
	)
}
