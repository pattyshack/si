package platform

type ArchitectureName string
type OperatingSystemName string

const (
	Amd64 = ArchitectureName("amd64")

	Linux = OperatingSystemName("linux")
)

type Platform interface {
	ArchitectureName() ArchitectureName
	OperatingSystemName() OperatingSystemName

	SysCallTypeSpec() SysCallTypeSpec

	ArchitectureRegisters() *ArchitectureRegisters
}
