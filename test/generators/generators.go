package generators

import (
	"github.com/Meerschwein/nixos-go-up/pkg/configuration"
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"pgregory.net/rapid"
)

func String(t *rapid.T, label string) string {
	return rapid.String().Draw(t, label).(string)
}

func Int(t *rapid.T, label string) int {
	return rapid.Int().Draw(t, label).(int)
}

func Bool(t *rapid.T, label string) bool {
	return rapid.Bool().Draw(t, label).(bool)
}

func DiskGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) disk.Disk {
		return disk.Disk{
			Name:             String(t, "Disk_Name"),
			SizeGB:           Int(t, "Disk_SizeGB"),
			Encrypt:          Bool(t, "Disk_Encrypt"),
			Yubikey:          Bool(t, "Disk_Yubikey"),
			EncryptionPasswd: String(t, "Disk_EncryptionPasswd"),
			PartitionTable:   rapid.SampledFrom([]disk.PartitionTable{disk.Gpt, disk.Mbr}).Draw(t, "Disk_Table").(disk.PartitionTable),
			Partitions:       rapid.SliceOfN(PartitionGen(), 0, 5).Draw(t, "Disk_Partitions").([]disk.Partition),
		}
	})
}

func PartitionGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) disk.Partition {
		return disk.Partition{
			Format:   rapid.SampledFrom([]disk.Filesystem{disk.Ext4, disk.Fat32}).Draw(t, "Partition_Format").(disk.Filesystem),
			Label:    String(t, "Partition_Label"),
			Primary:  Bool(t, "Partition_Primary"),
			Path:     String(t, "Partition_Path"),
			Number:   Int(t, "Partition_Number"),
			From:     String(t, "Partition_From"),
			To:       String(t, "Partition_To"),
			Bootable: Bool(t, "Partition_Bootable"),
		}
	})
}

func ConfigurationGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) configuration.Conf {
		return configuration.Conf{
			Disk:     DiskGen().Draw(t, "Disk").(disk.Disk),
			Hostname: String(t, "Hostname"),
			Timezone: String(t, "Timezone"),
			Username: String(t, "Username"),
			Password: String(t, "Password"),
			DesktopEnviroment: rapid.SampledFrom([]configuration.DesktopEnviroment{
				configuration.GNOME,
				configuration.XFCE,
				configuration.NONE,
			}).Draw(t, "DesktopEnviroment").(configuration.DesktopEnviroment),
			KeyboardLayout: String(t, "KeyboardLayout"),
		}
	})
}

func FirmwareGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) configuration.Firmware {
		return rapid.SampledFrom([]configuration.Firmware{configuration.UEFI, configuration.BIOS}).Draw(t, "Firmware").(configuration.Firmware)
	})
}
