package generators

import (
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
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
			Vendor:           String(t, "Disk_Vendor"),
			Model:            String(t, "Disk_Model"),
			SizeGB:           Int(t, "Disk_SizeGB"),
			Encrypt:          Bool(t, "Disk_Encrypt"),
			Yubikey:          Bool(t, "Disk_Yubikey"),
			EncryptionPasswd: String(t, "Disk_EncryptionPasswd"),
			Table:            rapid.SampledFrom([]disk.Table{disk.Gpt, disk.Mbr}).Draw(t, "Disk_Table").(disk.Table),
			Partitions:       rapid.SliceOfN(PartitionGen(), 0, 5).Draw(t, "Disk_Partitions").([]disk.Partition),
		}
	})
}

func PartitionGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) disk.Partition {
		return disk.Partition{
			Format:   rapid.SampledFrom([]disk.DiskFormat{disk.Ext4, disk.Fat32}).Draw(t, "Partition_Format").(disk.DiskFormat),
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

func SelectionGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) selection.Selection {
		return selection.Selection{
			Disk:              DiskGen().Draw(t, "Disk").(disk.Disk),
			Hostname:          String(t, "Hostname"),
			Timezone:          String(t, "Timezone"),
			Username:          String(t, "Username"),
			Password:          String(t, "Password"),
			DesktopEnviroment: rapid.SampledFrom([]selection.DesktopEnviroment{selection.GNOME, selection.XFCE, selection.NONE}).Draw(t, "DesktopEnviroment").(selection.DesktopEnviroment),
			KeyboardLayout:    String(t, "KeyboardLayout"),
		}
	})
}

func FirmwareGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) disk.Firmware {
		return rapid.SampledFrom([]disk.Firmware{disk.UEFI, disk.BIOS}).Draw(t, "Firmware").(disk.Firmware)
	})
}
