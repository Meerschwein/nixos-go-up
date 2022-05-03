package generators

import (
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"pgregory.net/rapid"
)

func DiskGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) disk.Disk {
		return disk.Disk{
			Name:       rapid.String().Draw(t, "Disk_Name").(string),
			Vendor:     rapid.String().Draw(t, "Disk_Vendor").(string),
			Model:      rapid.String().Draw(t, "Disk_Model").(string),
			SizeGB:     rapid.Int().Draw(t, "Disk_SizeGB").(int),
			Table:      rapid.SampledFrom([]disk.Table{disk.Gpt, disk.Mbr}).Draw(t, "Disk_Table").(disk.Table),
			Partitions: rapid.SliceOfN(PartitionGen(), 0, 5).Draw(t, "Disk_Partitions").([]disk.Partition),
		}
	})
}

func PartitionGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) disk.Partition {
		return disk.Partition{
			Format:   rapid.SampledFrom([]disk.DiskFormat{disk.Ext4, disk.Fat32}).Draw(t, "Partition_Format").(disk.DiskFormat),
			Label:    rapid.String().Draw(t, "Partition_Label").(string),
			Primary:  rapid.Bool().Draw(t, "Partition_Primary").(bool),
			Path:     rapid.String().Draw(t, "Partition_Path").(string),
			Number:   rapid.Int().Draw(t, "Partition_Number").(int),
			From:     rapid.String().Draw(t, "Partition_From").(string),
			To:       rapid.String().Draw(t, "Partition_To").(string),
			Bootable: rapid.Bool().Draw(t, "Partition_Bootable").(bool),
		}
	})
}

func SelectionGen() *rapid.Generator {
	return rapid.Custom(func(t *rapid.T) selection.Selection {
		return selection.Selection{
			Disk:              DiskGen().Draw(t, "Disk").(disk.Disk),
			Hostname:          rapid.String().Draw(t, "Hostname").(string),
			Timezone:          rapid.String().Draw(t, "Timezone").(string),
			Username:          rapid.String().Draw(t, "Username").(string),
			Password:          rapid.String().Draw(t, "Password").(string),
			DesktopEnviroment: rapid.SampledFrom([]selection.DesktopEnviroment{selection.GNOME, selection.XFCE}).Draw(t, "DesktopEnviroment").(selection.DesktopEnviroment),
			KeyboardLayout:    rapid.String().Draw(t, "KeyboardLayout").(string),
		}
	})
}
