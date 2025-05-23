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
	"regexp"
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

type Executable struct {
	URL  URL      `json:"url,omitempty"`
	Size int64    `json:"size,omitempty"`
	Hash Sha3V512 `json:"hash,omitempty"`
}

type Application struct {
	AppID        string       `json:"id"`
	InstID       string       `json:"instanceId"`
	Sandbox      Sandbox      `json:"sandbox"`
	Backup       Backup       `json:"backup"`
	Restore      Restore      `json:"restore"`
	Executable   Executable   `json:"executable"`
	Build        Build        `json:"build,omitzero"`
	ReverseProxy ReverseProxy `json:"reverseProxy,omitzero"`
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
	S3 S3 `json:"s3"`
	// Files which represent the entire application. This may include the binary itself and additional companion
	// files. The binary is the first file, where the executable flag is set.
	FileSet FileSet `json:"fileSet"`
}

type Sandbox struct {
	Unit       ServiceUnit `json:"systemd"`
	Filesystem Filesystem  `json:"filesystem"`
}

// Filesystem configuration
type Filesystem struct {
	Enabled bool `json:"enabled,omitempty"`
	// Maximum amount of usable persistent disk space in the context (e.g. a sandbox). This requires
	// a filesystem which supports usrquota,grpquota. Value will be set through setquota.
	UsrQuota Memory `json:"max"`
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Name string

func (n Name) Valid() bool {
	return nameRegex.MatchString(string(n))
}

// ServiceUnit contains all declarative systemd service sections. See also
// https://www.freedesktop.org/software/systemd/man/latest/systemd.resource-control.html
// and do not forget to audit sandboxing score by systemd-analyze security .
type ServiceUnit struct {
	Unit    UnitSection    `json:"unit"`
	Install InstallSection `json:"install"`
	Service ServiceSection `json:"service"`
}

type InstallSection struct {
	// e.g. multi-user.target
	WantedBy string `json:"wantedBy"`
}

type UnitSection struct {
	Description string `json:"description,omitempty"`
	// e.g. "network-online.target"
	After Name `json:"after,omitempty"`
}

// BindRule configures restrictions on the ability of unit processes to invoke bind(2) on a socket.
// Both allow and deny rules to be defined that restrict which addresses a socket may be bound to.
// See https://www.freedesktop.org/software/systemd/man/latest/systemd.resource-control.html.
type BindRule string

// ProtectProc takes one of "noaccess", "invisible", "ptraceable" or "default" (which it defaults to).
type ProtectProc string

// PrivateUsers is "yes", "self" or "identity"
type PrivateUsers string

// "yes", "private" or "strict"
type ProtectControlGroups string

type RestrictNamespaces string

// yes, "full" or "strict"
type ProtectSystem string

// yes, "read-only" or "tmpfs".
type ProtectHome string

// Restart is one of no, on-success, on-failure, on-abnormal, on-watchdog, on-abort, or always.
type Restart string

// Type is one of simple, exec, forking, oneshot, dbus, notify, notify-reload, or idle.
type Type string

// OOMPolicy is one of continue, stop or kill.
type OOMPolicy string

type CapabilityBoundingSet string

// Takes a space-separated combination of options from the following list: keep-caps, keep-caps-locked,
// no-setuid-fixup, no-setuid-fixup-locked, noroot, and noroot-locked.
type SecureBits string
type ServiceSection struct {

	// e.g. allow 1234 and 4321 but deny all
	SocketBindAllow []BindRule `json:"socketBindAllow,omitempty"`
	SocketBindDeny  []BindRule `json:"socketBindDeny,omitempty"`

	//When set, this controls the "hidepid=" mount option of the "procfs" instance for the unit that controls which
	//directories with process metainformation (/proc/PID) are visible and accessible: when set to "noaccess"
	//the ability to access most of other users' process metadata in /proc/ is taken away for processes of the service.
	//When set to "invisible" processes owned by other users are hidden from /proc/. If "ptraceable" all processes that
	//cannot be ptrace()'ed by a process are hidden to it.
	ProtectProc ProtectProc `json:"protectProc,omitempty"`
	//  If set, a UNIX user and group pair is allocated dynamically when the unit is started, and released as soon as
	// it is stopped. The user and group will not be added to /etc/passwd or /etc/group, but are
	//managed transiently during runtime.
	DynamicUser bool `json:"dynamicUser,omitempty"`

	// If set, all System V and POSIX IPC objects owned by the user and group the processes of this unit are run as
	// are removed when the unit is stopped. This setting only has an effect if at least one of User=, Group= and
	// DynamicUser= are used.
	RemoveIPC bool `json:"removeIPC,omitempty"`

	// If enabled, a new file system namespace will be set up for the executed processes, and /tmp/ and /var/tmp/
	// directories inside it are not shared with processes outside of the namespace, plus all temporary files
	// created by a service in these directories will be removed after the service is stopped.
	PrivateTmp bool `json:"privateTmp,omitempty"`

	// If true, sets up a new /dev/ mount for the executed processes and only adds API pseudo devices such as
	// /dev/null, /dev/zero or /dev/random (as well as the pseudo TTY subsystem) to it, but no physical devices
	// such as /dev/sda, system memory /dev/mem, system ports /dev/port and others.
	PrivateDevices bool `json:"privateDevices,omitempty"`

	// If true, sets up a new network namespace for the executed processes and configures only the loopback network device
	// "lo" inside it. No other network devices will be available to the executed process. This is useful to turn off
	// network access by the executed process.
	PrivateNetwork bool `json:"privateNetwork,omitempty"`

	// Takes a boolean argument. If true, sets up a new IPC namespace for the executed processes. Each IPC namespace
	// has its own set of System V IPC identifiers and its own POSIX message queue file system. This is useful
	// to avoid name clash of IPC identifiers. Defaults to false. It is possible to run two or more units within the
	// same private IPC namespace by using the JoinsNamespaceOf= directive, see systemd.unit(5) for details.
	PrivateIPC bool `json:"privateIPC,omitempty"`

	// Defaults to false. If enabled, sets up a new PID namespace for the executed processes. Each executed process
	// is now PID 1 - the init process - in the new namespace. /proc/ is mounted such that only processes in the
	// PID namespace are visible. If PrivatePIDs= is set, MountAPIVFS=yes is implied.
	PrivatePIDs bool `json:"privatePIDs,omitempty"`

	// Takes a boolean argument or one of "self" or "identity". Defaults to false. If enabled, sets up a new user
	//namespace for the executed processes and configures a user and group mapping. If set to a true value or "self",
	//a minimal user and group mapping is configured that maps the "root" user and group as well as the unit's own
	//user and group to themselves and everything else to the "nobody" user and group. This is useful to securely
	//detach the user and group databases used by the unit from the rest of the system, and thus to create an effective
	//sandbox environment.
	PrivateUsers PrivateUsers `json:"privateUsers,omitempty"`

	//  If set, writes to the hardware clock or system clock will be denied. Defaults to off.
	ProtectClock bool `json:"protectClock,omitempty"`

	// If true, kernel variables accessible through /proc/sys/, /sys/, /proc/sysrq-trigger,
	// /proc/latency_stats, /proc/acpi, /proc/timer_stats, /proc/fs and /proc/irq will be made read-only and
	// /proc/kallsyms as well as /proc/kcore will be inaccessible to all processes of the unit.
	ProtectKernelTunables bool `json:"protectKernelTunables,omitempty"`

	// If true, explicit module loading will be denied. This allows module load and unload operations
	// to be turned off on modular kernels. It is recommended to turn this on for most services that do not
	// need special file systems or extra kernel modules to work.
	ProtectKernelModules bool `json:"protectKernelModules,omitempty"`

	// If true, access to the kernel log ring buffer will be denied. It is recommended to turn this on for
	// most services that do not need to read from or write to the kernel log ring buffer.
	ProtectKernelLogs bool `json:"protectKernelLogs,omitempty"`

	// Takes a boolean argument. When set, sets up a new UTS namespace for the executed processes.
	// In addition, changing hostname or domainname is prevented. Defaults to off.
	ProtectHostname bool `json:"protectHostname,omitempty"`

	// Takes a boolean argument or the special values "private" or "strict". If true, the Linux Control Groups
	// (cgroups(7)) hierarchies accessible through /sys/fs/cgroup/ will be made read-only to all processes of the unit.
	// If set to "private", the unit will run in a cgroup namespace with a private writable mount of /sys/fs/cgroup/.
	// If set to "strict", the unit will run in a cgroup namespace with a private read-only mount of /sys/fs/cgroup/.
	// Defaults to off. If ProtectControlGroups= is set, MountAPIVFS=yes is implied. Note "private" and "strict" are
	// downgraded to false and true respectively unless the system is using the unified control group hierarchy and the
	// kernel supports cgroup namespaces.
	//
	// Except for container managers no services should require write access to the control groups hierarchies;
	// it is hence recommended to set ProtectControlGroups= to true or "strict" for most services. For this setting
	// the same restrictions regarding mount propagation and privileges apply as for ReadOnlyPaths= and related
	// settings, see above.
	ProtectControlGroups ProtectControlGroups `json:"protectControlGroups,omitempty"`

	// Restricts access to Linux namespace functionality for the processes of this unit. For details about Linux
	// namespaces, see namespaces(7). Either takes a boolean argument, or a space-separated list of namespace
	// type identifiers. If false (the default), no restrictions on namespace creation and switching are made. If
	// true, access to any kind of namespacing is prohibited. Otherwise, a space-separated list of namespace type
	// identifiers must be specified, consisting of any combination of: cgroup, ipc, net, mnt, pid, user and uts.
	RestrictNamespaces []RestrictNamespaces `json:"restrictNamespaces,omitempty"`

	// If set, attempts to create memory mappings that are writable and executable at the same time, or to change
	// existing memory mappings to become executable, or mapping shared memory segments as executable, are prohibited.
	// Specifically, a system call filter is added (or preferably, an equivalent kernel check is enabled with prctl(2))
	// that rejects mmap(2) system calls with both PROT_EXEC and PROT_WRITE set, mprotect(2) or pkey_mprotect(2) system
	// calls with PROT_EXEC set and shmat(2) system calls with SHM_EXEC set. Note that this option is incompatible
	// with programs and libraries that generate program code dynamically at runtime, including JIT execution engines,
	// executable stacks, and code "trampoline" feature of various C compilers.
	MemoryDenyWriteExecute bool `json:"memoryDenyWriteExecute,omitempty"`

	// If set, any attempts to enable realtime scheduling in a process of the unit are refused. This restricts access
	// to realtime task scheduling policies such as SCHED_FIFO, SCHED_RR or SCHED_DEADLINE. See sched(7) for
	// details about these scheduling policies. Realtime scheduling policies may be used to monopolize CPU time for
	// longer periods of time, and may hence be used to lock up or otherwise trigger Denial-of-Service situations on
	// the system. It is hence recommended to restrict access to realtime scheduling to the few programs that
	// actually require them.
	RestrictRealtime bool `json:"restrictRealtime,omitempty"`

	// Takes a boolean argument. If set, any attempts to set the set-user-ID (SUID) or set-group-ID (SGID) bits on
	// files or directories will be denied (for details on these bits see inode(7)). As the SUID/SGID bits are
	// mechanisms to elevate privileges, and allow users to acquire the identity of other users, it is recommended
	// to restrict creation of SUID/SGID files to the few programs that actually require them. Note that this
	// restricts marking of any type of file system object with these bits, including both regular files and
	// directories (where the SGID is a different meaning than for files, see documentation). This option is
	// implied if DynamicUser= is enabled.
	RestrictSUIDSGID bool `json:"restrictSUIDSGID,omitempty"`

	// If set, the processes of this unit will be run in their own private file system (mount) namespace with
	// all mount propagation from the processes towards the host's main file system namespace turned off. This means
	// any file system mount points established or removed by the unit's processes will be private to them and not be
	// visible to the host. However, file system mount points established or removed on the host will be
	// propagated to the unit's processes.
	PrivateMounts bool `json:"privateMounts,omitempty"`

	// Takes a space-separated list of system call names. If this setting is used, all system calls executed
	// by the unit processes except for the listed ones will result in immediate process termination with the
	// SIGSYS signal (allow-listing).
	SystemCallFilter string `json:"systemCallFilter,omitempty"`

	// This option may be specified more than once, in which case all listed variables will be set. If the same
	// variable is listed twice, the later setting will override the earlier setting. If the empty string is
	// assigned to this option, the list of environment variables is reset, all prior assignments have no effect.
	//
	// The names of the variables can contain ASCII letters, digits, and the underscore character. Variable names
	// cannot be empty or start with a digit. In variable values, most characters are allowed, but non-printable
	// characters are currently rejected.
	//
	// Note that environment variables are not suitable for passing secrets (such as passwords, key material, …)
	// to service processes. Environment variables set for a unit are exposed to unprivileged clients via D-Bus IPC,
	// and generally not understood as being data that requires protection. Moreover, environment variables are
	// propagated down the process tree, including across security boundaries (such as setuid/setgid executables),
	// and hence might leak to processes that should not have access to the secret data.
	// Use LoadCredential=, LoadCredentialEncrypted= or SetCredentialEncrypted= (see below) to pass
	// data to unit processes securely.
	Environment []EnvVar `json:"environment,omitempty"`

	// Takes a boolean argument or the special values "full" or "strict". If true, mounts the /usr/ and the boot
	// loader directories (/boot and /efi) read-only for processes invoked by this unit. If set to "full", the
	// /etc/ directory is mounted read-only, too. If set to "strict" the entire file system hierarchy is mounted
	// read-only, except for the API file system subtrees /dev/, /proc/ and /sys/ (protect these directories using
	// PrivateDevices=, ProtectKernelTunables=, ProtectControlGroups=). This setting ensures that any modification
	// of the vendor-supplied operating system (and optionally its configuration, and local mounts) is prohibited for
	// the service. It is recommended to enable this setting for all long-running services, unless they are involved
	// with system updates or need to modify the operating system in other ways.
	//
	// Note that if ProtectSystem= is set to "strict" and PrivateTmp= is enabled, then /tmp/ and /var/tmp/
	// will be writable.
	ProtectSystem ProtectSystem `json:"protectSystem,omitempty"`

	// Takes a boolean argument or the special values "read-only" or "tmpfs". If true, the directories /home/, /root,
	// and /run/user are made inaccessible and empty for processes invoked by this unit. If set to "read-only", the
	// three directories are made read-only instead. If set to "tmpfs", temporary file systems are mounted on the three
	// directories in read-only mode. The value "tmpfs" is useful to hide home directories not relevant to the processes
	// invoked by the unit, while still allowing necessary directories to be made visible when listed in BindPaths=
	// or BindReadOnlyPaths=.
	//
	// Setting this to "yes" is mostly equivalent to setting the three directories in InaccessiblePaths=. Similarly,
	// "read-only" is mostly equivalent to ReadOnlyPaths=, and "tmpfs" is mostly equivalent to
	// TemporaryFileSystem= with ":ro".
	//
	// It is recommended to enable this setting for all long-running services (in particular network-facing ones),
	// to ensure they cannot get access to private user data, unless the services actually require access to the
	// user's private data. This setting is implied if DynamicUser= is set. This setting cannot ensure protection
	// in all cases. In general it has the same limitations as ReadOnlyPaths=, see below.
	//
	// This option is only available for system services, or for services running in per-user instances of the
	// service manager in which case PrivateUsers= is implicitly enabled (requires unprivileged user namespaces
	// support to be enabled in the kernel via the "kernel.unprivileged_userns_clone=" sysctl).
	ProtectHome ProtectHome `json:"protectHome,omitempty"`

	// These options take a whitespace-separated list of directory names.
	// The specified directory names must be relative, and may not include "..". If set,
	// when the unit is started, one or more directories by the specified names will be created
	// (including their parents) below the locations defined in the following table. Also, the corresponding
	// environment variable will be defined with the full paths of the directories. If multiple directories are set,
	// then in the environment variable the paths are concatenated with colon (":").
	//
	// Example: /var/lib/	$XDG_STATE_HOME	$STATE_DIRECTORY
	//
	// Example: if a system service unit has the following,
	//
	// RuntimeDirectory=foo/bar
	// StateDirectory=aaa/bbb ccc
	//
	// then the environment variable "RUNTIME_DIRECTORY" is set with "/run/foo/bar", and "STATE_DIRECTORY"
	// is set with "/var/lib/aaa/bbb:/var/lib/ccc".
	StateDirectory string `json:"stateDirectory,omitempty"`

	// Sets up a new file system namespace for executed processes. These options may be used to limit access a process
	// has to the file system. Each setting takes a space-separated list of paths relative to the host's root directory
	// (i.e. the system running the service manager). Note that if paths contain symlinks, they are resolved relative
	// to the root directory set with RootDirectory=/RootImage=.
	ExecPaths         string `json:"execPaths,omitempty"`
	ReadOnlyPaths     string `json:"readOnlyPaths,omitempty"`
	ReadWritePaths    string `json:"readWritePaths,omitempty"`
	InaccessiblePaths string `json:"inaccessiblePaths,omitempty"`

	// Configures unit-specific bind mounts. A bind mount makes a particular file or directory available at an
	// additional place in the unit's view of the file system. Any bind mounts created with this option are
	// specific to the unit, and are not visible in the host's mount table. This option expects a whitespace
	// separated list of bind mount definitions. Each definition consists of a colon-separated triple of source
	// path, destination path and option string, where the latter two are optional. If only a source path is
	// specified the source and destination is taken to be the same. The option string may be either "rbind" or
	// "norbind" for configuring a recursive or non-recursive bind mount. If the destination path is omitted, the
	// option string must be omitted too. Each bind mount definition may be prefixed with "-", in which case it
	// will be ignored when its source path does not exist.
	BindPaths         string `json:"bindPaths,omitempty"`
	BindReadOnlyPaths string `json:"bindReadOnlyPaths,omitempty"`

	// Configures whether the service shall be restarted when the service process exits, is killed, or a timeout
	// is reached. The service process may be the main service process, but it may also be one of the processes
	// specified with ExecStartPre=, ExecStartPost=, ExecStop=, ExecStopPost=, or ExecReload=. When the death of
	// the process is a result of systemd operation (e.g. service stop or restart), the service will not be restarted.
	// Timeouts include missing the watchdog "keep-alive ping" deadline and a service start, reload, and
	// stop operation timeouts.
	Restart Restart `json:"restart,omitempty"`

	// Type is huge, see https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html#Type=.
	// Probably favor exec over simple.
	Type Type `json:"type,omitempty"`

	// Commands that are executed when this service is started.
	ExecStart Command `json:"execStart"`

	// Configures the time to sleep before restarting a service (as configured with Restart=).
	// Takes a unit-less value in seconds, or a time span value such as "5min 20s". Defaults to 100ms.
	RestartSec time.Duration `json:"restartSec,omitempty"`

	// Sets the adjustment value for the Linux kernel's Out-Of-Memory (OOM) killer score for executed processes.
	// Takes an integer between -1000 (to disable OOM killing of processes of this unit) and 1000 (to make
	// killing of processes of this unit under memory pressure very likely). See The /proc Filesystem for details.
	// If not specified defaults to the OOM score adjustment level of the service manager itself,
	// which is normally at 0.
	OOMPolicy OOMPolicy `json:"OOMPolicy,omitempty"`

	OOMScoreAdjust int `json:"OOMScoreAdjust,omitempty"`

	// These options are only available for system services and are not supported for services running in
	// per-user instances of the service manager.
	// When used in conjunction with DynamicUser= the user/group name specified is dynamically
	// allocated at the time the service is started, and released at the time the service is stopped — unless it
	// is already allocated statically (see below). If DynamicUser= is not used the specified user and group must
	// have been created statically in the user database no later than the moment the service is started,
	// for example using the sysusers.d(5) facility, which is applied at boot or package install time.
	// If the user does not exist by then program invocation will fail.
	User  string `json:"user,omitempty"`
	Group string `json:"group,omitempty"`

	// Takes a boolean parameter that controls whether to set the $HOME, $LOGNAME, and $SHELL environment variables.
	// If not set, this defaults to true if User=, DynamicUser= or PAMName= are set, false otherwise. If set to true,
	// the variables will always be set for system services, i.e. even when the default user "root" is used.
	// If set to false, the mentioned variables are not set by the service manager, no matter whether
	// User=, DynamicUser=, or PAMName= are used or not. This option normally has no effect on services of
	// the per-user service manager, since in that case these variables are typically inherited from user
	// manager's own environment anyway.
	SetLoginEnvironment bool `json:"setLoginEnvironment,omitempty"`

	// Controls which capabilities to include in the capability bounding set for the executed process.
	// See capabilities(7) for details. Takes a whitespace-separated list of capability names,
	// e.g. CAP_SYS_ADMIN, CAP_DAC_OVERRIDE, CAP_SYS_PTRACE. Capabilities listed will be included
	// in the bounding set, all others are removed. If the list of capabilities is prefixed with "~",
	// all but the listed capabilities will be included, the effect of the assignment inverted. Note that
	// this option also affects the respective capabilities in the effective, permitted and inheritable capability
	// sets. If this option is not used, the capability bounding set is not modified on process execution,
	// hence no limits on the capabilities of the process are enforced. This option may appear more than once,
	// in which case the bounding sets are merged by OR, or by AND if the lines are prefixed with "~" (see below).
	// If the empty string is assigned to this option, the bounding set is reset to the empty capability set, and
	// all prior settings have no effect. If set to "~" (without any further argument), the bounding set is reset
	// to the full set of available capabilities, also undoing any previous settings. This does not affect
	// commands prefixed with "+".
	CapabilityBoundingSet []CapabilityBoundingSet `json:"capabilityBoundingSet,omitempty"`

	// If true, ensures that the service process and all its children can never gain new privileges through execve()
	// (e.g. via setuid or setgid bits, or filesystem capabilities). This is the simplest and most effective way
	// to ensure that a process and its children can never elevate privileges again. Defaults to false. In case the
	// service will be run in a new mount namespace anyway and SELinux is disabled, all file systems are mounted
	// with MS_NOSUID flag. Also see No New Privileges Flag.
	//
	// Note that this setting only has an effect on the unit's processes themselves (or any processes directly
	// or indirectly forked off them). It has no effect on processes potentially invoked on request of them
	// through tools such as at(1), crontab(1), systemd-run(1), or arbitrary IPC services.
	NoNewPrivileges bool `json:"noNewPrivileges,omitempty"`

	// Controls the secure bits set for the executed process. Takes a space-separated combination of options from
	// the following list: keep-caps, keep-caps-locked, no-setuid-fixup, no-setuid-fixup-locked, noroot, and
	// noroot-locked. This option may appear more than once, in which case the secure bits are ORed.
	// If the empty string is assigned to this option, the bits are reset to 0. This does not affect
	// commands prefixed with "+". See capabilities(7) for details.
	SecureBits []SecureBits `json:"secureBits,omitempty"`

	// Takes a profile name as argument. The process executed by the unit will switch to this profile when started.
	// Profiles must already be loaded in the kernel, or the unit will fail. If prefixed by "-", all errors
	// will be ignored. This setting has no effect if AppArmor is not enabled. This setting does not
	// affect commands prefixed with "+".
	AppArmorProfile string `json:"appArmorProfile,omitempty"`

	// These settings control the memory controller in the unified hierarchy.
	//
	//Specify the throttling limit on memory usage of the executed processes in this unit. Memory usage may go above
	// the limit if unavoidable, but the processes are heavily slowed down and memory is taken away aggressively
	// in such cases. This is the main mechanism to control memory usage of a unit.
	//
	//
	// Alternatively, a percentage value may be specified, which is taken relative to the installed physical
	// memory on the system. If assigned the special value "infinity", no memory throttling is applied.
	// This controls the "memory.high" control group attribute. For details about this control group attribute,
	// see Memory Interface Files. The effective configuration is reported as
	// EffectiveMemoryHigh= (see also EffectiveMemoryMax=).
	//
	// While StartupMemoryHigh= applies to the startup and shutdown phases of the system, MemoryHigh= applies
	// to normal runtime of the system, and if the former is not set also to the startup and shutdown phases.
	// Using StartupMemoryHigh= allows prioritizing specific services at boot-up and shutdown differently
	// than during normal runtime.
	MemoryHigh        Memory `json:"memoryHigh,omitempty"`
	StartupMemoryHigh Memory `json:"startupMemoryHigh,omitempty"`

	// These settings control the memory controller in the unified hierarchy.
	//
	// Specify the absolute limit on swap usage of the executed processes in this unit.
	//
	// Takes a swap size in bytes. If the value is suffixed with K, M, G or T, the specified swap size is parsed
	// as Kilobytes, Megabytes, Gigabytes, or Terabytes (with the base 1024), respectively. Alternatively, a
	// percentage value may be specified, which is taken relative to the specified swap size on the system.
	// If assigned the special value "infinity", no swap limit is applied. These settings control the
	// "memory.swap.max" control group attribute. For details about this control group attribute,
	// see Memory Interface Files.
	//
	// While StartupMemorySwapMax= applies to the startup and shutdown phases of the system,
	// MemorySwapMax= applies to normal runtime of the system, and if the former is not set
	// also to the startup and shutdown phases. Using StartupMemorySwapMax= allows prioritizing
	// specific services at boot-up and shutdown differently than during normal runtime.
	MemorySwapMax        Memory `json:"memorySwapMax,omitempty"`
	StartupMemorySwapMax Memory `json:"startupMemorySwapMax,omitempty"`

	// If set to an integer value, assign the specified CPU time weight to the processes executed, if the
	// unified control group hierarchy is used on the system. These options control the "cpu.weight" control
	// group attribute. The allowed range is 1 to 10000. Defaults to unset, but the kernel default is 100.
	CPUWeight int `json:"CPUWeight,omitempty"`

	// This setting controls the cpu controller in the unified hierarchy.
	//
	// Assign the specified CPU time quota to the processes executed. Takes a percentage value, suffixed with "%".
	// The percentage specifies how much CPU time the unit shall get at maximum, relative to the total CPU time
	// available on one CPU. Use values > 100% for allotting CPU time on more than one CPU. This controls the
	// "cpu.max" attribute on the unified control group hierarchy and "cpu.cfs_quota_us" on legacy.
	// For details about these control group attributes, see Control Groups v2 and CFS Bandwidth Control.
	// Setting CPUQuota= to an empty value unsets the quota.
	//
	// Example: CPUQuota=20% ensures that the executed processes will never get more than 20% CPU time on one CPU.
	CPUQuota int `json:"CPUQuota,omitempty"`

	// Specifies how processes of this unit shall be killed. One of control-group, mixed, process, none.
	// If set to control-group, all remaining processes in the control group of this unit will be killed on unit stop
	// (for services: after the stop command is executed, as configured with ExecStop=). If set to mixed, the SIGTERM
	// signal (see below) is sent to the main process while the subsequent SIGKILL signal (see below) is sent to all
	// remaining processes of the unit's control group. If set to process, only the main process itself is killed
	// (not recommended!). If set to none, no process is killed (strongly recommended against!). In this case, only
	// the stop command will be executed on unit stop, but no process will be killed otherwise. Processes remaining
	// alive after stop are left in their control group and the control group continues to exist after stop unless empty.
	//
	// Note that it is not recommended to set KillMode= to process or even none, as this allows processes to escape
	// the service manager's lifecycle and resource management, and to remain running even while their service is
	// considered stopped and is assumed to not consume any resources.
	//
	// Defaults to control-group.
	KillMode KillMode `json:"killMode,omitempty"`

	// Specifies which signal to use when stopping a service. This controls the signal that is sent as first step of
	// shutting down a unit (see above), and is usually followed by SIGKILL (see above and below).
	// For a list of valid signals, see signal(7). Defaults to SIGTERM.
	KillSignal KillSignal `json:"killSignal,omitempty"`
}

// KillSignal is one of the standard signals like SIGKILL, SIGTERM etc
type KillSignal string

// KillMode is one of control-group, mixed, process, none.
type KillMode string

// Memory is a a memory size in bytes. If the value is suffixed with K, M, G or T, the specified memory size is parsed
// as Kilobytes, Megabytes, Gigabytes, or Terabytes (with the base 1024), respectively.
type Memory string

type URL string

type Command struct {
	Cmd  string   `json:"cmd,omitempty"`
	Args []string `json:"args,omitempty"`
}

type EnvVar struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
