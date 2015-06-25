package dockerclient

import (
	"io"
)

type Callback func(*Event, chan error, ...interface{})

type StatCallback func(string, *Stats, chan error, ...interface{})

type Client interface {
	Info() (*Info, error)
	ListContainers(all, size bool, filters string) ([]Container, error)
	InspectContainer(id string) (*ContainerInfo, error)
	InspectImage(id string) (*ImageInfo, error)
	CreateContainer(config *ContainerConfig, name string) (string, error)
	ContainerLogs(id string, options *LogOptions) (io.ReadCloser, error)
	ContainerChanges(id string) ([]*ContainerChanges, error)
	// ContainerStats takes an optional stop channel and returns a
	// StatsOrError channel. If an error is ever sent, then no more stats
	// will be sent on that channel. If a stop channel is provided, events
	// will stop being monitored after the stop channel is closed.
	ContainerStats(id string, stopChan <-chan struct{}) (<-chan StatsOrError, error)
	Exec(config *ExecConfig) (string, error)
	StartContainer(id string, config *HostConfig) error
	AttachContainer(id string, options *AttachOptions) (io.ReadCloser, error)
	StopContainer(id string, timeout int) error
	RestartContainer(id string, timeout int) error
	KillContainer(id, signal string) error
	// MonitorEvents takes options and an optional stop channel and returns
	// an EventOrError channel. If an error is ever sent, then no more
	// events will be sent. If a stop channel is provided, events will stop
	// being monitored after the stop channel is closed.
	MonitorEvents(options *MonitorEventsOptions, stopChan <-chan struct{}) (<-chan EventOrError, error)
	StartMonitorEvents(cb Callback, ec chan error, args ...interface{})
	StopAllMonitorEvents()
	StartMonitorStats(id string, cb StatCallback, ec chan error, args ...interface{})
	StopAllMonitorStats()
	TagImage(nameOrID string, repo string, tag string, force bool) error
	Version() (*Version, error)
	PullImage(name string, auth *AuthConfig, cliOut io.Writer) error
	LoadImage(reader io.Reader) error
	RemoveContainer(id string, force, volumes bool) error
	ListImages() ([]*Image, error)
	RemoveImage(name string) ([]*ImageDelete, error)
	PauseContainer(name string) error
	UnpauseContainer(name string) error
	RenameContainer(oldName string, newName string) error
	ImportImage(source string, repository string, tag string, tar io.Reader) (io.ReadCloser, error)
}