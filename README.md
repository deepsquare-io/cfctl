# cfctl

_A command-line bootstrapping and management tool for [k0s zero friction kubernetes](https://k0sproject.io/) clusters._

- [Installation](#installation)
- [Development status](#development-status)
- [Usage](#usage)
- [Configuration](#configuration-file)

Example output of cfctl deploying a k0s cluster:

```sh
INFO ==> Running phase: Connect to hosts
INFO ==> Running phase: Detect host operating systems
INFO [ssh] 10.0.0.1:22: is running Ubuntu 20.10
INFO [ssh] 10.0.0.2:22: is running Ubuntu 20.10
INFO ==> Running phase: Prepare hosts
INFO ==> Running phase: Gather host facts
INFO [ssh] 10.0.0.1:22: discovered 10.12.18.133 as private address
INFO ==> Running phase: Validate hosts
INFO ==> Running phase: Gather k0s facts
INFO ==> Running phase: Download k0s binaries on hosts
INFO ==> Running phase: Configure k0s
INFO ==> Running phase: Initialize the k0s cluster
INFO [ssh] 10.0.0.1:22: installing k0s controller
INFO ==> Running phase: Install workers
INFO [ssh] 10.0.0.1:22: generating token
INFO [ssh] 10.0.0.2:22: installing k0s worker
INFO [ssh] 10.0.0.2:22: waiting for node to become ready
INFO ==> Running phase: Disconnect from hosts
INFO ==> Finished in 2m2s
INFO k0s cluster version 1.22.3+k0s.0 is now installed
INFO Tip: To access the cluster you can now fetch the admin kubeconfig using:
INFO      cfctl kubeconfig
```

You can find example Terraform and [bootloose](https://github.com/k0sproject/bootloose) configurations in the [examples/](examples/) directory.

## Installation

### Install from the released binaries

Download the desired version for your operating system and processor architecture from the [cfctl releases page](https://github.com/SquareFactory/cfctl/releases). Make the file executable and place it in a directory available in your `$PATH`.

As the released binaries aren't signed yet, on macOS and Windows, you must first run the executable via "Open" in the context menu and allow running it.

### Install from the sources

If you have a working Go toolchain, you can use `go install` to install cfctl to your `$GOPATH/bin`.

```sh
go install github.com/SquareFactory/cfctl@latest
```

#### Shell auto-completions

##### Bash

```sh
cfctl completion > /etc/bash_completion.d/cfctl
```

##### Zsh

```sh
cfctl completion > /usr/local/share/zsh/site-functions/_cfctl
```

##### Fish

```sh
cfctl completion > ~/.config/fish/completions/cfctl.fish
```

## Anonymous telemetry

cfctl sends anonymized telemetry data when it is used. This can be disabled via the `--disable-telemetry` flag or by setting the environment variable `DISABLE_TELEMETRY=true`.

The telemetry data includes:

- cfctl version
- Operating system + CPU architecture ("linux x86", "darwin arm64", ...)
- An anonymous machine ID generated by [denisbrodbeck/machineid](https://github.com/denisbrodbeck/machineid) or if that fails, an md5 sum of the hostname
- Event information:
  - Phase name ("Connecting to hosts", "Gathering facts", ...) and the duration how long it took to finish
  - Cluster UUID (`kubectl get -n kube-system namespace kube-system -o template={{.metadata.uid}}`)
  - Was k0s dynamic config enabled (true/false)
  - Was a custom or the default k0s configuration used (true/false)
  - In case of a crash, a backtrace with source filenames and line numbers only

The data is used to estimate the number of users and to identify failure hotspots.

## Development status

cfctl is ready for use and in continuous development. It is still at a stage where maintaining backwards compatibility is not a high priority goal.

Missing major features include at least:

- The released binaries have not been signed
- The configuration specification and command-line interface options are still evolving

## Usage

### `cfctl apply`

The main function of cfctl is the `cfctl apply` subcommand. Provided a configuration file describing the desired cluster state, cfctl will connect to the listed hosts, determines the current state of the hosts and configures them as needed to form a k0s cluster.

The default location for the configuration file is `cfctl.yaml` in the current working directory. To load a configuration from a different location, use:

```sh
cfctl apply --config path/to/cfctl.yaml
```

If the configuration cluster version `spec.k0s.version` is greater than the version detected on the cluster, a cluster upgrade will be performed. If the configuration lists hosts that are not part of the cluster, they will be configured to run k0s and will be joined to the cluster.

### `cfctl init`

Generate a configuration template. Use `--k0s` to include an example `spec.k0s.config` k0s configuration block. You can also supply a list of host addresses via arguments or stdin.

Output a minimal configuration template:

```sh
cfctl init > cfctl.yaml
```

Output an example configuration with a default k0s config:

```sh
cfctl init --k0s > cfctl.yaml
```

Create a configuration from a list of host addresses and pipe it to cfctl apply:

```sh
cfctl init 10.0.0.1 10.0.0.2 ubuntu@10.0.0.3:8022 | cfctl apply --config -
```

### `cfctl backup & restore`

Takes a [backup](https://docs.k0sproject.io/main/backup/) of the cluster control plane state into the current working directory.

The files are currently named with a running (unix epoch) timestamp, e.g. `k0s_backup_1623220591.tar.gz`.

Restoring a backup can be done as part of the [cfctl apply](#cfctl-apply) command using `--restore-from k0s_backup_1623220591.tar.gz` flag.

Restoring the cluster state is a full restoration of the cluster control plane state, including:

- Etcd datastore content
- Certificates
- Keys

In general restore is intended to be used as a disaster recovery mechanism and thus it expects that no k0s components actually exist on the controllers.

Known limitations in the current restore process:

- The control plane address (`externalAddress`) needs to remain the same between backup and restore. This is caused by the fact that all worker node components connect to this address and cannot currently be re-configured.

### `cfctl reset`

Uninstall k0s from the hosts listed in the configuration.

### `cfctl kubeconfig`

Connects to the cluster and outputs a kubeconfig file that can be used with `kubectl` or `kubeadm` to manage the kubernetes cluster.

Example:

```sh
$ cfctl kubeconfig --config path/to/cfctl.yaml > k0s.config
$ kubectl get node --kubeconfig k0s.config
NAME      STATUS     ROLES    AGE   VERSION
worker0   NotReady   <none>   10s   v1.20.2-k0s1
```

## Configuration file

The configuration file is in YAML format and loosely resembles the syntax used in Kubernetes. YAML anchors and aliases can be used.

To generate a simple skeleton configuration file, you can use the `cfctl init` subcommand.

Configuration example:

```yaml
apiVersion: cfctl.clusterfactory.io/v1beta1
kind: Cluster
metadata:
  name: my-k0s-cluster
spec:
  hosts:
    - role: controller
      installFlags:
        - --debug
      ssh:
        address: 10.0.0.1
        user: root
        port: 22
        keyPath: ~/.ssh/id_rsa
    - role: worker
      installFlags:
        - --debug
      ssh:
        address: 10.0.0.2
  k0s:
    version: 0.10.0
    config:
      apiVersion: k0s.k0sproject.io/v1beta1
      kind: ClusterConfig
      metadata:
        name: my-k0s-cluster
      spec:
        images:
          calico:
            cni:
              image: calico/cni
              version: v3.16.2
```

### Environment variable substitution

Simple bash-like expressions are supported in the configuration for environment variable substition.

- `$VAR` or `${VAR}` value of `VAR` environment variable
- `${var:-DEFAULT_VALUE}` will use `VAR` if non-empty, otherwise `DEFAULT_VALUE`
- `$$var` - escape, result will be `$var`.
- And [several other expressions](https://github.com/a8m/envsubst#docs)

### Configuration Header Fields

###### `apiVersion` &lt;string&gt; (required)

The configuration file syntax version. Currently the only supported version is `cfctl.clusterfactory.io/v1beta1`.

###### `kind` &lt;string&gt; (required)

In the future, some of the configuration APIs can support multiple types of objects. For now, the only supported kind is `Cluster`.

###### `spec` &lt;mapping&gt; (required)

The main object definition, see [below](#configuration-spec)

###### `metadata` &lt;mapping&gt; (optional)

Information that can be used to uniquely identify the object.

Example:

```yaml
metadata:
  name: k0s-cluster-name
```

### Spec Fields

##### `spec.hosts` &lt;sequence&gt; (required)

A list of cluster hosts. Host requirements:

- Currently only linux targets are supported
- The user must either be root or have passwordless `sudo` access.
- The host must fulfill the k0s system requirements

See [host object documentation](#host-fields) below.

##### `spec.k0s` &lt;mapping&gt; (optional)

Settings related to the k0s cluster.

See [k0s object documentation](#k0s-fields) below.

### Host Fields

###### `spec.hosts[*].role` &lt;string&gt; (required)

One of:

- `controller` - a controller host
- `controller+worker` - a controller host that will also run workloads
- `single` - a [single-node cluster](https://docs.k0sproject.io/main/k0s-single-node/) host, the configuration can only contain one host
- `worker` - a worker host

###### `spec.hosts[*].noTaints` &lt;boolean&gt; (optional) (default: `false`)

When `true` and used in conjuction with the `controller+worker` role, the default taints are disabled making regular workloads schedulable on the node. By default, k0s sets a node-role.kubernetes.io/master:NoSchedule taint on controller+worker nodes and only workloads with toleration for it will be scheduled.

###### `spec.hosts[*].uploadBinary` &lt;boolean&gt; (optional) (default: `false`)

When `true`, the k0s binaries for target host will be downloaded and cached on the local host and uploaded to the target.
When `false`, the k0s binary downloading is performed on the target host itself

###### `spec.hosts[*].k0sBinaryPath` &lt;string&gt; (optional)

A path to a file on the local host that contains a k0s binary to be uploaded to the host. Can be used to test drive a custom development build of k0s.

###### `spec.hosts[*].hostname` &lt;string&gt; (optional)

Override host's hostname. When not set, the hostname reported by the operating system is used.

###### `spec.hosts[*].dataDir` &lt;string&gt; (optional) (default: `/var/lib/k0s`)

Set host's k0s data-dir.

###### `spec.hosts[*].installFlags` &lt;sequence&gt; (optional)

Extra flags passed to the `k0s install` command on the target host. See `k0s install --help` for a list of options.

###### `spec.hosts[*].environment` &lt;mapping&gt; (optional)

List of key-value pairs to set to the target host's environment variables.

Example:

```yaml
environment:
  HTTP_PROXY: 10.0.0.1:443
```

###### `spec.hosts[*].files` &lt;sequence&gt; (optional)

List of files to be uploaded to the host.

Example:

```yaml
- name: image-bundle
  src: airgap-images.tgz
  dstDir: /var/lib/k0s/images/
  perm: 0600
```

- `name`: name of the file "bundle", used only for logging purposes (optional)
- `src`: File path, an URL or [Glob pattern](https://golang.org/pkg/path/filepath/#Match) to match files to be uploaded. URL sources will be directly downloaded using the target host (required)
- `dstDir`: Destination directory for the file(s). `cfctl` will create full directory structure if it does not already exist on the host (default: user home)
- `dst`: Destination filename for the file. Only usable for single file uploads (default: basename of file)
- `perm`: File permission mode for uploaded file(s) (default: same as local)
- `dirPerm`: Directory permission mode for created directories (default: 0755)
- `user`: User name of file/directory owner, must exist on the host (optional)
- `group`: Group name of file/directory owner, must exist on the host (optional)

###### `spec.hosts[*].hooks` &lt;mapping&gt; (optional)

Run a set of commands on the remote host during cfctl operations.

Example:

```yaml
hooks:
  apply:
    before:
      - date >> cfctl-apply.log
    after:
      - echo "apply success" >> cfctl-apply.log
```

The currently available "hook points" are:

- `apply`: Runs during `cfctl apply`
  - `before`: Runs after configuration and host validation, right before configuring k0s on the host
  - `after`: Runs before disconnecting from the host after a successful apply operation
- `backup`: Runs during `k0s backup`
  - `before`: Runs before cfctl runs the `k0s backup` command
  - `after`: Runs before disconnecting from the host after successfully taking a backup
- `reset`: Runs during `cfctl reset`
  - `before`: Runs after gathering information about the cluster, right before starting to remove the k0s installation.
  - `after`: Runs before disconnecting from the host after a successful reset operation

##### `spec.hosts[*].os` &lt;string&gt; (optional) (default: ``)

Override OS distribution auto-detection. By default `cfctl` detects the OS by reading `/etc/os-release` or `/usr/lib/os-release` files. In case your system is based on e.g. Debian but the OS release info has something else configured you can override `cfctl` to use Debian based functionality for the node with:

```yaml
- role: worker
  os: debian
  ssh:
    address: 10.0.0.2
```

##### `spec.hosts[*].privateInterface` &lt;string&gt; (optional) (default: ``)

Override private network interface selected by host fact gathering.
Useful in case fact gathering picks the wrong private network interface.

```yaml
- role: worker
  os: debian
  privateInterface: eth1
```

##### `spec.hosts[*].privateAddress` &lt;string&gt; (optional) (default: ``)

Override private IP address selected by host fact gathering.
Useful in case fact gathering picks the wrong IPAddress.

```yaml
- role: worker
  os: debian
  privateAddress: 10.0.0.2
```

##### `spec.hosts[*].ssh` &lt;mapping&gt; (optional)

SSH connection options.

Example:

```yaml
spec:
  hosts:
    - role: controller
      ssh:
        address: 10.0.0.2
        user: ubuntu
        keyPath: ~/.ssh/id_rsa
```

It's also possible to tunnel connections through a bastion host. The bastion configuration has all the same fields as any SSH connection:

```yaml
spec:
  hosts:
    - role: controller
      ssh:
        address: 10.0.0.2
        user: ubuntu
        keyPath: ~/.ssh/id_rsa
        bastion:
          address: 10.0.0.1
          user: root
          keyPath: ~/.ssh/id_rsa2
```

SSH agent and auth forwarding are also supported, a host without a keyfile:

```yaml
spec:
  hosts:
    - role: controller
      ssh:
        address: 10.0.0.2
        user: ubuntu
```

```shell
$ ssh-add ~/.ssh/aws.pem
$ ssh -A user@jumphost
user@jumphost ~ $ cfctl apply
```

Pageant or openssh-agent can be used on Windows.

###### `spec.hosts[*].ssh.address` &lt;string&gt; (required)

IP address of the host

###### `spec.hosts[*].ssh.user` &lt;string&gt; (optional) (default: `root`)

Username to log in as.

###### `spec.hosts[*].ssh.port` &lt;number&gt; (required)

TCP port of the SSH service on the host.

###### `spec.hosts[*].ssh.keyPath` &lt;string&gt; (optional) (default: `~/.ssh/identity ~/.ssh/id_rsa ~/.ssh/id_dsa`)

Path to an SSH key file. If a public key is used, ssh-agent is required. When left empty, the default value will first be looked for from the ssh configuration (default `~/.ssh/config`) `IdentityFile` parameter.

##### `spec.hosts[*].localhost` &lt;mapping&gt; (optional)

Localhost connection options. Can be used to use the local host running cfctl as a node in the cluster.

###### `spec.hosts[*].localhost.enabled` &lt;boolean&gt; (optional) (default: `false`)

This must be set `true` to enable the localhost connection.

##### `spec.hosts[*].openSSH` &lt;mapping&gt; (optional)

An alternative SSH client protocol that uses the system's openssh client for connections.

Example:

```yaml
spec:
  hosts:
    - role: controller
      openSSH:
        address: 10.0.0.2
```

The only required field is the `address` and it can also be a hostname that is found in the ssh config. All other options such as user, port and keypath will use the same defaults as if running `ssh` from the command-line or will use values found from the ssh config.

An example SSH config:

```
Host controller1
  Hostname 10.0.0.1
  Port 2222
  IdentityFile ~/.ssh/id_cluster_esa
```

If this is in your `~/.ssh/config`, you can simply use the host alias as the address in your cfctl config:

```yaml
spec:
  hosts:
    - role: controller
      openSSH:
        address: controller1
        # if the ssh configuration is in a different file, you can use:
        # configPath: /path/to/config
```

###### `spec.hosts[*].openSSH.address` &lt;string&gt; (required)

IP address, hostname or ssh config host alias of the host

###### `spec.hosts[*].openSSH.user` &lt;string&gt; (optional)

Username to connect as.

###### `spec.hosts[*].openSSH.port` &lt;number&gt; (optional)

Remote port.

###### `spec.hosts[*].openSSH.keyPath` &lt;string&gt; (optional)

Path to private key.

###### `spec.hosts[*].openSSH.configPath` &lt;string&gt; (optional)

Path to ssh config, defaults to ~/.ssh/config with fallback to /etc/ssh/ssh_config.

###### `spec.hosts[*].openSSH.disableMultiplexing` &lt;boolean&gt; (optional)

The default mode of operation is to use connection multiplexing where a ControlMaster connection is opened and the subsequent connections to the same host use the master connection over a socket to communicate to the host.

If this is disabled by setting `disableMultiplexing: true`, running every remote command will require reconnecting and reauthenticating to the host.

###### `spec.hosts[*].openSSH.options` &lt;mapping&gt; (optional)

Additional options as key/value pairs to use when running the ssh client.

Example:

```yaml
openSSH:
  address: host
  options:
    ForwardAgent: true # -o ForwardAgent=yes
    StrictHostkeyChecking: false # -o StrictHostkeyChecking: no
```

###### `spec.hosts[*].reset` &lt;boolean&gt; (optional) (default: `false`)

If set to `true` cfctl will remove the node from kubernetes and reset k0s on the host.

### K0s Fields

##### `spec.k0s.version` &lt;string&gt; (optional) (default: auto-discovery)

The version of k0s to deploy. When left out, cfctl will default to using the latest released version of k0s or the version already running on the cluster.

##### `spec.k0s.versionChannel` &lt;string&gt; (optional) (default: `stable`)

Possible values are `stable` and `latest`.

When `spec.k0s.version` is left undefined, this setting can be set to `latest` to allow cfctl to include k0s pre-releases when looking for the latest version. The default is to only look for stable releases.

##### `spec.k0s.dynamicConfig` &lt;boolean&gt; (optional) (default: false)

Enable k0s dynamic config. The setting will be automatically set to true if:

- Any controller node has `--enable-dynamic-config` in `installFlags`
- Any existing controller node has `--enable-dynamic-config` in run arguments (`k0s status -o json`)

**Note:**: When running k0s in dynamic config mode, cfctl will ONLY configure the cluster-wide configuration during the first time initialization, after that the configuration has to be managed via `k0s config edit` or `cfctl config edit`. The node specific configuration will be updated on each apply.

See also:

- [k0s Dynamic Configuration](https://docs.k0sproject.io/main/dynamic-configuration/)

##### `spec.k0s.config` &lt;mapping&gt; (optional) (default: auto-generated)

Embedded k0s cluster configuration. See [k0s configuration documentation](https://docs.k0sproject.io/main/configuration/) for details.

When left out, the output of `k0s config create` will be used.
