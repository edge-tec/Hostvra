package osadapter

import (
	"errors"
)

var (
	ErrUnsupportedOS = errors.New("unsupported operating system distribution")
)

type ServiceAction string

const (
	ActionStart   ServiceAction = "start"
	ActionStop    ServiceAction = "stop"
	ActionRestart ServiceAction = "restart"
	ActionReload  ServiceAction = "reload"
	ActionEnable  ServiceAction = "enable"
	ActionDisable ServiceAction = "disable"
	ActionStatus  ServiceAction = "status"
)

type OSAdapter interface {
	Family() string
	Name() string
	Version() string
	Architecture() string
	KernelVersion() string
	PackageManager() string
	ServiceAction(service string, action ServiceAction) error
	IsServiceRunning(service string) (bool, error)
	GetConfigPath(service string) string
}

type BaseOSInfo struct {
	OSName         string
	OSVersion      string
	OSFamily       string
	Arch           string
	Kernel         string
	PkgManager     string
}

func (b *BaseOSInfo) Name() string           { return b.OSName }
func (b *BaseOSInfo) Version() string        { return b.OSVersion }
func (b *BaseOSInfo) Family() string         { return b.OSFamily }
func (b *BaseOSInfo) Architecture() string   { return b.Arch }
func (b *BaseOSInfo) KernelVersion() string  { return b.Kernel }
func (b *BaseOSInfo) PackageManager() string { return b.PkgManager }
