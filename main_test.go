package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	behavior := os.Getenv("TEST_BEHAVIOR")
	switch behavior {
	case "":
		os.Exit(m.Run())
	case "mockDf":
		dfMockResult()
	default:
		log.Fatalf("unknown behavior %q", behavior)
	}
}

func Test_isSupportedFsType(t *testing.T) {
	type args struct {
		fstype string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"nfs", args{"nfs"}, true},
		{"cifs", args{"cifs"}, true},
		{"ext4", args{"ext4"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSupportedFsType(tt.args.fstype); got != tt.want {
				t.Errorf("isSupportedFsType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMountPoints(t *testing.T) {
	want := []string{"/mnt/nfs-data"}
	got := getMountPoints("test/example_fstab")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getMountPoints() = %v, want %v", got, want)
	}
}

func dfMockResult() {
	fmt.Println(`Filesystem                                     Type       1024-blocks       Used     Available Capacity Mounted on
devtmpfs                                       devtmpfs       7843444          0       7843444       0% /dev
tmpfs                                          tmpfs          7866136          0       7866136       0% /dev/shm
/dev/mapper/vg_group-appdata                   ext4           67067908    7405684      59662224      12% /appdata
192.168.1.137:/                                nfs      1099511627776 1354400768 1098157227008       1% /mnt/nfs
//192.168.1.112/share                          cifs        5368709120  113656064    5255053056       3% /mnt/cifs
				`)
}

func copyExecutable(t *testing.T, src, dst string) {
	i, err := os.Open(src)
	require.NoError(t, err, "cannot open %q", src)
	defer i.Close()

	o, err := os.Create(dst)
	require.NoError(t, err, "cannot create %q", dst)
	defer o.Close()

	_, err = io.Copy(o, i)
	require.NoError(t, err, "cannot copy binary")

	require.NoError(t, os.Chmod(dst, 0o755), "cannot update permissions")
}

func Test_getMountedPoints(t *testing.T) {
	testExe, err := os.Executable()
	require.NoError(t, err, "can't determine current exectuable")

	binDir := t.TempDir()
	newPath := binDir + string(filepath.ListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", newPath)

	dfExe := filepath.Join(binDir, "df")
	copyExecutable(t, testExe, dfExe)

	t.Setenv("TEST_BEHAVIOR", "mockDf")

	mountPoints := getMountedPoints()
	expected := []string{"/mnt/nfs", "/mnt/cifs"}
	assert.Equal(t, expected, mountPoints)
}
