//go:build windows

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

type diskFree struct {
	freeBytes int64
}

func windowsGetDiskFreeSpace(path string) (*diskFree, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable int64
	var totalBytes int64
	var totalFreeBytes int64

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	_, _, e := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if e != nil && e != syscall.Errno(0) {
		return nil, e
	}

	return &diskFree{freeBytes: freeBytesAvailable}, nil
}

type detectedDrive struct {
	Letter    string
	Path      string
	FreeGB    float64
	IsCloud   bool
	CloudType string
	Label     string
}

// detectDrives scans all drive letters and cloud markers
func detectDrives() []detectedDrive {
	var drives []detectedDrive

	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"} {
		root := letter + ":\\"
		if _, err := os.Stat(root); err != nil {
			continue
		}

		d := detectedDrive{Letter: letter, Path: root}

		if fd, err := windowsGetDiskFreeSpace(root); err == nil {
			d.FreeGB = float64(fd.freeBytes) / (1024 * 1024 * 1024)
		}

		d.IsCloud, d.CloudType = detectCloudDrive(root)
		if d.IsCloud {
			d.Label = root + " (" + d.CloudType + ")"
		}

		drives = append(drives, d)
	}

	return drives
}

// detectCloudDrive checks filesystem markers for cloud providers
func detectCloudDrive(root string) (bool, string) {
	if _, err := os.Stat(filepath.Join(root, "My Drive")); err == nil {
		return true, "google_drive"
	}
	if _, err := os.Stat(filepath.Join(root, ".Encrypted")); err == nil {
		return true, "google_drive"
	}
	if _, err := os.Stat(filepath.Join(root, ".dropbox")); err == nil {
		return true, "dropbox"
	}
	if _, err := os.Stat(filepath.Join(root, "Box")); err == nil {
		return true, "box"
	}
	if _, err := os.Stat(filepath.Join(root, "pCloud")); err == nil {
		return true, "pcloud"
	}
	if _, err := os.Stat(filepath.Join(root, "iCloud Drive")); err == nil {
		return true, "icloud"
	}
	if _, err := os.Stat(filepath.Join(root, "MEGAsync")); err == nil {
		return true, "mega"
	}
	if _, err := os.Stat(filepath.Join(root, "MEGA")); err == nil {
		return true, "mega"
	}
	if _, err := os.Stat(filepath.Join(root, "Nextcloud")); err == nil {
		return true, "nextcloud"
	}
	if _, err := os.Stat(filepath.Join(root, ".stfolder")); err == nil {
		return true, "syncthing"
	}

	return false, ""
}

// ---------- Provider detection ----------

// detectGoogleDriveMount finds the actual Google Drive sync folder.
// Only returns real sync folders — never cache dirs (they don't sync upstream).
func detectGoogleDriveMount() string {
	// 1. Drive letters with "My Drive" subfolder
	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"} {
		root := letter + ":\\"
		if _, err := os.Stat(root); err != nil {
			continue
		}
		myDrive := filepath.Join(root, "My Drive")
		if info, err := os.Stat(myDrive); err == nil && info.IsDir() {
			return myDrive
		}
	}

	// 2. User home folder mount
	home, _ := os.UserHomeDir()
	for _, name := range []string{"Google Drive", "GoogleDrive", "Google Drive File Stream", "GDFS"} {
		p := filepath.Join(home, name)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}

	// 3. Registry-configured mount points (admin policy)
	for _, sk := range []string{
		`Software\Policies\Google\DriveFS`,
		`Software\Policies\Google\DriveFS\Share`,
		`Software\Google\DriveFS`,
		`Software\Google\DriveFS\Share`,
	} {
		if k, err := registry.OpenKey(registry.CURRENT_USER, sk, registry.READ); err == nil {
			for _, val := range []string{"DefaultMountPoint", "BasePath"} {
				if mountPoint, _, err := k.GetStringValue(val); err != nil {
					mountPoint = os.ExpandEnv(mountPoint)
					if info, err := os.Stat(mountPoint); err == nil && info.IsDir() {
						k.Close()
						return mountPoint
					}
				}
			}
			k.Close()
		}
		if k, err := registry.OpenKey(registry.LOCAL_MACHINE, sk, registry.READ); err == nil {
			if mountPoint, _, err := k.GetStringValue("DefaultMountPoint"); err != nil {
				mountPoint = os.ExpandEnv(mountPoint)
				if info, err := os.Stat(mountPoint); err == nil && info.IsDir() {
					k.Close()
					return mountPoint
				}
			}
			k.Close()
		}
	}

	return ""
}

// detectOneDriveFolder finds OneDrive sync folder
func detectOneDriveFolder() string {
	if v := os.Getenv("OneDrive"); v != "" {
		if info, err := os.Stat(v); err == nil && info.IsDir() {
			return v
		}
	}

	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\OneDrive\Accounts\Personal`, registry.READ); err == nil {
		if userFolder, _, err := k.GetStringValue("UserFolder"); err == nil {
			if info, err := os.Stat(userFolder); err == nil && info.IsDir() {
				k.Close()
				return userFolder
			}
		}
		k.Close()
	}

	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, "OneDrive")
	if info, err := os.Stat(fallback); err == nil && info.IsDir() {
		return fallback
	}

	return ""
}

// detectDropboxFolder finds Dropbox sync folder.
// Official approach: read %LOCALAPPDATA%\Dropbox\info.json → personal.path
func detectDropboxFolder() string {
	// 1. info.json (official Dropbox API — recommended)
	localAppData := os.Getenv("LOCALAPPDATA")
	for _, base := range []string{
		filepath.Join(localAppData, "Dropbox", "info.json"),
		filepath.Join(os.Getenv("APPDATA"), "Dropbox", "info.json"),
	} {
		if data, err := os.ReadFile(base); err == nil {
			var info struct {
				Personal struct {
					Path string `json:"path"`
				} `json:"personal"`
			}
			if json.Unmarshal(data, &info) == nil && info.Personal.Path != "" {
				if stat, err := os.Stat(info.Personal.Path); err == nil && stat.IsDir() {
					return info.Personal.Path
				}
			}
		}
	}

	// 2. Registry SyncRootManager (non-default locations)
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\SyncRootManager`, registry.READ); err == nil {
		subkeys, _ := k.ReadSubKeyNames(-1)
		for _, sk := range subkeys {
			if strings.Contains(strings.ToLower(sk), "dropbox") {
				if subk, err := registry.OpenKey(k, sk+`\UserSyncRoots`, registry.READ); err == nil {
					vals, _, _ := subk.GetStringsValue("1")
					for _, v := range vals {
						if info, err := os.Stat(v); err == nil && info.IsDir() {
							subk.Close()
							k.Close()
							return v
						}
					}
					subk.Close()
				}
			}
		}
		k.Close()
	}

	// 3. Home dir fallback
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, "Dropbox")
	if info, err := os.Stat(fallback); err == nil && info.IsDir() {
		return fallback
	}

	return ""
}

// detectBoxFolder finds Box Drive sync folder.
// Default: %USERPROFILE%\Box, admin-configurable via HKLM\SOFTWARE\Box\Box\CustomBoxLocation
func detectBoxFolder() string {
	// 1. Registry custom location
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Box\Box`, registry.READ); err == nil {
		if custom, _, err := k.GetStringValue("CustomBoxLocation"); err == nil {
			custom = os.ExpandEnv(custom)
			boxPath := filepath.Join(custom, "Box")
			if info, err := os.Stat(boxPath); err == nil && info.IsDir() {
				k.Close()
				return boxPath
			}
			if info, err := os.Stat(custom); err == nil && info.IsDir() {
				k.Close()
				return custom
			}
		}
		k.Close()
	}

	// 2. Default location
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, "Box")
	if info, err := os.Stat(fallback); err == nil && info.IsDir() {
		return fallback
	}

	return ""
}

// detectPCloudFolder finds pCloud Drive mount point.
// Default: %USERPROFILE%\pCloudDrive (virtual FUSE drive)
func detectPCloudFolder() string {
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, "pCloudDrive")
	if info, err := os.Stat(fallback); err == nil && info.IsDir() {
		return fallback
	}

	// Check if mounted as a drive letter
	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"} {
		root := letter + ":\\"
		if _, err := os.Stat(root); err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "pCloud")); err == nil {
			return root
		}
	}

	return ""
}

// detectICloudFolder finds iCloud Drive sync folder.
// Default: %USERPROFILE%\iCloud Drive, configurable in iCloud app v14+
func detectICloudFolder() string {
	home, _ := os.UserHomeDir()
	for _, name := range []string{"iCloud Drive", "iCloudDrive"} {
		p := filepath.Join(home, name)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}

	// Check LOCALAPPDATA for Apple iCloud
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		p := filepath.Join(localAppData, "Apple", "iCloud Drive")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}

	return ""
}

// detectMEGAFolder finds MEGAsync sync folder.
// Default: %USERPROFILE%\Documents\MEGAsync
func detectMEGAFolder() string {
	home, _ := os.UserHomeDir()

	// Default location
	for _, name := range []string{
		filepath.Join("Documents", "MEGAsync"),
		filepath.Join("Documents", "MEGA"),
		"MEGAsync",
		"MEGA",
	} {
		p := filepath.Join(home, name)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}

	return ""
}

// detectNextcloudFolder finds Nextcloud sync folder.
// Default: %USERPROFILE%\Nextcloud, configurable in nextcloud.cfg
func detectNextcloudFolder() string {
	// 1. Parse nextcloud.cfg for configured folder paths
	appData := os.Getenv("APPDATA")
	if appData != "" {
		cfgPath := filepath.Join(appData, "Nextcloud", "nextcloud.cfg")
		if data, err := os.ReadFile(cfgPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "folder") && strings.Contains(line, "path=") {
					idx := strings.Index(line, "path=")
					if idx >= 0 {
						val := line[idx+5:]
						val = strings.Trim(val, "\"' ")
						val = os.ExpandEnv(val)
						if info, err := os.Stat(val); err == nil && info.IsDir() {
							return val
						}
					}
				}
			}
		}
	}

	// 2. Default location
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, "Nextcloud")
	if info, err := os.Stat(fallback); err == nil && info.IsDir() {
		return fallback
	}

	return ""
}

// detectSyncthingFolder finds Syncthing default folder.
// Config at %LOCALAPPDATA%\Syncthing\config.xml, default path is ~ (home dir)
func detectSyncthingFolder() string {
	// 1. Check if default "Sync" folder exists in home
	home, _ := os.UserHomeDir()
	syncFolder := filepath.Join(home, "Sync")
	if info, err := os.Stat(syncFolder); err == nil && info.IsDir() {
		// Verify it's a Syncthing folder (.stfolder marker)
		if _, err := os.Stat(filepath.Join(syncFolder, ".stfolder")); err == nil {
			return syncFolder
		}
	}

	// 2. Check for .stfolder in home subdirectories
	entries, _ := os.ReadDir(home)
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(home, e.Name())
			if _, err := os.Stat(filepath.Join(p, ".stfolder")); err == nil {
				return p
			}
		}
	}

	// 3. Check drive letters for .stfolder
	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"} {
		root := letter + ":\\"
		if _, err := os.Stat(root); err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, ".stfolder")); err == nil {
			return root
		}
	}

	return ""
}

// detectNetworkDrives finds mapped network drives and SMB shares
func detectNetworkDrives() []detectedDrive {
	var drives []detectedDrive

	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"} {
		root := letter + ":\\"
		volName := getVolumeName(root)
		if strings.HasPrefix(volName, "\\\\") {
			d := detectedDrive{
				Letter:    letter,
				Path:      root,
				IsCloud:   true,
				CloudType: "network",
				Label:     root + " (" + volName + ")",
			}
			if fd, err := windowsGetDiskFreeSpace(root); err == nil {
				d.FreeGB = float64(fd.freeBytes) / (1024 * 1024 * 1024)
			}
			drives = append(drives, d)
		}
	}

	return drives
}

func getVolumeName(path string) string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getVolumeName := kernel32.NewProc("GetVolumeInformationW")

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}

	var nameBuf [256]uint16
	_, _, e := getVolumeName.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&nameBuf[0])),
		uintptr(len(nameBuf)),
		0, 0, 0, 0, 0,
	)
	if e != nil && e != syscall.Errno(0) {
		return ""
	}

	return syscall.UTF16ToString(nameBuf[:])
}

func detectGitHubCLI() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// findProviderRoot returns the sync folder root for a given provider name.
func findProviderRoot(provider string) string {
	switch provider {
	case "google_drive":
		return detectGoogleDriveMount()
	case "onedrive":
		return detectOneDriveFolder()
	case "dropbox":
		return detectDropboxFolder()
	case "box":
		return detectBoxFolder()
	case "pcloud":
		return detectPCloudFolder()
	case "icloud":
		return detectICloudFolder()
	case "mega":
		return detectMEGAFolder()
	case "nextcloud":
		return detectNextcloudFolder()
	case "syncthing":
		return detectSyncthingFolder()
	}
	return ""
}
