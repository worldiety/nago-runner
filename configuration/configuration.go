// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

// Package configuration contains the entire declarative model to describe all applications within a specific runner.
package configuration

import (
	"errors"
	"strings"
	"time"
)

// Runner describes all applications which this runner needs to provision.
type Runner struct {
	Applications []Application `json:"applications"`
}

// Backup describes also the secrets to backup the data into. If a runner is removed or compromised, the backup
// may get purged, encrypted or tampered as well. We may introduce another layer using our central app-console, but that
// does not scale and does not work if the console-server is down. Thus, if you need additional security
// against attacks to the backup, consider to backup the S3 as well. At least, a huge advantage is the scaling
// and configurability to the entire offsite storage system. Note that there are also object retention policies in
// some S3 implementations. The used S3 paths are prefixed, so that one bucket can be shared between multiple targets.
type Backup struct {
	Enabled  bool `json:"enabled,omitempty"`
	S3       S3   `json:"s3"`
	KeepDays int  `json:"keepDays,omitempty"`
}

type S3 struct {
	Enabled bool `json:"enabled,omitempty"`
	// e.g. "fsn1.your-objectstorage.com"
	Endpoint  string `json:"endpoint,omitempty"`
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
}

// Restore describes in a declarative way if data has to be restored from a backup. If the BackupID of the BasedOn
// state changes or the last mode changed,
// all current data of the data directory must be discarded and replaced by the backup.
type Restore struct {
	Enabled bool `json:"enabled,omitempty"`

	// RemoveExtra deletes any other files not declared in Files.
	RemoveExtra bool `json:"removeExtra,omitempty"`

	// Files to restore from.
	FileSet FileSetID `json:"fileSetId,omitempty"`

	// If ApplyAfter is in the past (relative to time.Now) and this restore configuration
	// has not yet been applied (based on internal state tracking), then the restore will be executed.
	ApplyAfter time.Time `json:"applyAfter"`
}

// A Path for a file in whatever context it must be interpreted. Probably absolute in the sandbox for example
// a root-based data dir like /data/mydata.tdb or /files/a/b/c.bin. A path containing '.' or '..' is invalid and will
// be rejected.
type Path string

func (p Path) Validate() error {
	if strings.Contains(string(p), ".") {
		return errors.New("path cannot contain '.'")
	}

	return nil
}

// Sha3V512 represents the hex encoded sha3 512 hashsum.
type Sha3V512 string

type FileSetID string
type FileSet struct {
	// ID is unique for all file sets.
	ID FileSetID `json:"ID,omitempty"`
	// Name of the FileSet, useful for identification, caching or referencing.
	Name string `json:"name,omitempty"`
	// Actual files within this file set.
	Files []File `json:"files,omitempty"`
	// Hash of all file hashes and sizes in alphabetical order.
	Hash Sha3V512 `json:"hash,omitempty"`
}

// A File must be a regular file and is never a hardlink or a softlink. It does not carry any permissions.
// It is intended to be restored within the context of a sandbox.
type File struct {
	Path Path     `json:"path,omitempty"`
	URL  URL      `json:"url,omitempty"`
	Size int64    `json:"size,omitempty"`
	Hash Sha3V512 `json:"hash,omitempty"`
	// Executable is true to indicate an (ELF) binary.
	Executable bool `json:"executable,omitempty"`
}

type ApplicationID string

type Application struct {
	State            State         `json:"state"`
	ID               ApplicationID `json:"id"`
	OrganizationSlug string        `json:"organizationSlug,omitempty"`
	ApplicationSlug  string        `json:"applicationSlug,omitempty"`
	Sandbox          Sandbox       `json:"sandbox"`
	Backup           Backup        `json:"backup"`
	Restore          Restore       `json:"restore"`
	Artifacts        Artifacts     `json:"artifacts"`
	Build            Build         `json:"build,omitzero"`
	ReverseProxy     ReverseProxy  `json:"reverseProxy,omitzero"`
}

type ReverseProxy struct {
	Enabled bool   `json:"enabled,omitempty"`
	Rules   []Rule `json:"rules"`
}

type Rule struct {
	// Location is like myapp.com or myapp.mycompany.nago.app
	Location Domain `json:"location,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	// If redirect is true, this does not apply proxy pass rules, but instead applies a http redirect
	Redirect       bool   `json:"redirect,omitempty"`
	RedirectTarget string `json:"redirectTarget,omitempty"`
}

type Domain string

type Build struct {
	Enabled bool   `json:"enabled,omitempty"`
	Git     Git    `json:"git,omitempty"`
	PureGo  PureGo `json:"pureGo,omitzero"`
}

// PureGo describes how to build a no-cgo purified gc-build of an application. This disables any
// code execution, and it allows most aggressive optimizations of the entire build process. It does not execute
// any other tests, vets, scripts or commands. Thus, it is best suited for rapid prototyping. It is executed
// on the runner itself to allow horizontal scaling of the infrastructure. The output file is named after
// directory name of the given imported MainPkg. In the example below, it would be just 'service'.
type PureGo struct {
	Enabled bool   `json:"enabled,omitempty"`
	MainPkg string `json:"mainPkg"` // e.g. gitlab.worldiety.net/mycompany/my-super-app/cmd/service
}

// Git describes how to access a git VCS repository.
type Git struct {
	URL           string `json:"url,omitempty"`    // e.g. git@gitlab.worldiety.net:mycompany/my-super-app.git
	Branch        string `json:"branch,omitempty"` // e.g. main
	SSHPrivateKey string `json:"sshPrivateKey,omitempty"`
	SSHPublicKey  string `json:"sshPublicKey,omitempty"`
}

// Artifacts from a build which contains at least a single executable file. The actual file data can be obtained
// from the declared S3 which may or not may be in the same bucket as the backups.
type Artifacts struct {
	State State `json:"state,omitempty"`
	S3    S3    `json:"s3"`
	// Files which represent the entire application. This may include the binary itself and additional companion
	// files. The binary is the first file, where the executable flag is set.
	FileSet FileSet `json:"fileSet"`
}

type Sandbox struct {
	Systemd    Systemd    `json:"systemd"`
	Filesystem Filesystem `json:"filesystem"`
}

// Filesystem configuration
type Filesystem struct {
	Enabled bool `json:"enabled,omitempty"`
	// Maximum amount of usable persistent disk space in the context (e.g. a sandbox). This requires
	// a filesystem which supports usrquota,grpquota. Value will be set through setquota.
	UsrQuota MemoryMiB `json:"max"`
}

type Systemd struct {
	Name   string `json:"name"`
	State  State  `json:"state"`
	NSpawn NSpawn `json:"NSpawn"`

	// e.g. control-group
	KillMode     string      `json:"killMode,omitempty"`
	TimeoutStart DurationSec `json:"timeoutStart,omitempty"`
	// e.g. SIGTERM
	KillSignal string `json:"killSignal,omitempty"`

	// security
	// e.g. ~CAP_SYS_ADMIN CAP_SETUID CAP_SETGID CAP_NET_ADMIN
	CapabilityBoundingSet string `json:"capabilityBoundingSet,omitempty"`
	NoNewPrivileges       bool   `json:"noNewPrivileges,omitempty"`
	// e.g. strict
	ProtectSystem string `json:"protectSystem,omitempty"`
	ProtectHome   bool   `json:"protectHome,omitempty"`
	PrivateTmp    bool   `json:"privateTmp,omitempty"`

	// cgroup
	MemoryMax MemoryMiB `json:"memoryMax,omitempty"`
	CPUQuota  Percent   `json:"CPUQuota,omitempty"`
}

type DebootstrapID string

type Debootstrap struct {
	ID    DebootstrapID `json:"id"`
	State State         `json:"state,omitempty"`
	// minbase|buildd|fakechroot, you probably want minbase
	Variant string `json:"variant,omitempty"`
	// e.g. plucky
	Suite string `json:"suite,omitempty"`
	// e.g. "http://ports.ubuntu.com/ubuntu-ports/"
	Mirror          URL       `json:"mirror,omitempty"`
	PostCommands    []Command `json:"commands,omitempty"`
	UpgradeCommands []Command `json:"upgradeCommands,omitempty"`
}

func (d Debootstrap) Target() Path {
	return "/var/lib/machines/" + Path(d.ID)
}

type URL string

type Command struct {
	Cmd  string   `json:"cmd,omitempty"`
	Args []string `json:"args,omitempty"`
}

// DurationSec is a duration in seconds.
type DurationSec int

// MemoryMiB is in Megabyte (1024*1024 byte)
type MemoryMiB int

// Percent is between 0 and 100
type Percent int

type NSpawn struct {
	Enabled bool `json:"enabled,omitempty"`
	// some readable machine name, e.g. myorg-myservice
	Machine string `json:"machine,omitempty"`
	// Sandbox root directory like /var/lib/machines/my_debootstrap_image
	Debootstrap Debootstrap `json:"debootstrap,omitzero"`
	// Additional Env key=value entries. By convention PORT (e.g. 3001), HOME (e.g. /data), TMPDIR (e.g. /tmp)
	// should be set.
	Envs []EnvVar `json:"envs,omitempty"`

	// Sets the working directory
	ChDir string `json:"chDir,omitempty"`

	// BindMounts binds the given paths into the container, like bind=/srv/builds/app123:/app
	BindMounts []BindMount `json:"bindMounts,omitempty"`

	// e.g. pick
	PrivateUsers string `json:"privateUsers,omitempty"`
}

// BindMount describes a host-container directory binding.
type BindMount struct {
	Host      Path `json:"host"`
	Container Path `json:"container"`
	ReadOnly  bool `json:"readOnly,omitempty"` // --bind-ro
}

type EnvVar struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type State string

const (
	// Absent state will remove any exiting bits, which are related to the given declaration.
	Absent State = "absent"
	// Present will add any required bits to match the given declaration.
	Present State = "present"
	// Disabled is the default and any associated declaration is ignored, thus no bits are touched.
	Disabled State = ""
)
