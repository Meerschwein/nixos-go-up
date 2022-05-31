package disk_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/test/generators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestDisk_PartitionName_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := generators.Disk().Draw(t, "Disk").(disk.Disk)
		part := rapid.Int().Draw(t, "Partition number").(int)

		res := d.PartitionName(part)

		require.True(t, strings.HasSuffix(res, strconv.Itoa(part)), "Partition number at the end")
		require.True(t, strings.HasPrefix(res, d.Name), "Diskname at the start")
	})
}

func TestDisk_PartitionName_Unit(t *testing.T) {
	testcases := []struct {
		Name      string
		Partition int
		Expected  string
	}{
		{
			Name:      "sda",
			Partition: 1,
			Expected:  "sda1",
		},
		{
			Name:      "sd",
			Partition: 2,
			Expected:  "sd2",
		},
		{
			Name:      "sdb",
			Partition: 3,
			Expected:  "sdb3",
		},
		{
			Name:      "nvme0n1",
			Partition: 1,
			Expected:  "nvme0n1p1",
		},
		{
			Name:      "nvme1n4",
			Partition: 2,
			Expected:  "nvme1n4p2",
		},
		{
			Name:      "nvme",
			Partition: 5,
			Expected:  "nvmep5",
		},
		{
			Name:      "test",
			Partition: 1,
			Expected:  "test1",
		},
		{
			Name:      "blahblah",
			Partition: 5,
			Expected:  "blahblah5",
		},
	}

	for _, test := range testcases {
		d := disk.Disk{Name: test.Name}

		actual := d.PartitionName(test.Partition)

		assert.Equal(t, test.Expected, actual)
	}
}
