package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coff33ninja/ai-memory/internal/memory"
	"github.com/coff33ninja/ai-memory/internal/persona"
	"github.com/coff33ninja/ai-memory/internal/version"
)

func handleBackupConfig(store *memory.Store, args map[string]interface{}) (interface{}, error) {
	provider, _ := args["provider"].(string)
	if provider == "" {
		provider = "local"
	}
	localPath, _ := args["local_path"].(string)
	autoBackup, _ := args["auto_backup"].(bool)
	intervalHours := 24
	if v, ok := args["interval_hours"].(float64); ok {
		intervalHours = int(v)
	}

	if provider == "local" && localPath == "" {
		return nil, fmt.Errorf("local provider requires local_path. Run list_backup_drives to see available drives, then set local_path to e.g. 'D:\\ai-memory-backups'")
	}

	// Validate writable
	if provider == "local" {
		testDir := filepath.Join(localPath, ".write-test")
		if err := os.MkdirAll(testDir, 0o755); err != nil {
			return nil, fmt.Errorf("cannot write to %s: %w", localPath, err)
		}
		os.Remove(testDir)
		os.Remove(filepath.Dir(testDir))
	}

	c, err := store.SetBackupConfig(provider, localPath, autoBackup, intervalHours)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Backup configured: provider=%s, path=%s, auto=%v, interval=%dh", c.Provider, c.LocalPath, c.AutoBackup, c.IntervalHours), nil
}

func handleListBackupDrives() (interface{}, error) {
	var sb strings.Builder
	sb.WriteString("Available backup locations:\n\n")

	seen := make(map[string]bool)

	// Cloud providers detected via registry/home dir
	type providerInfo struct {
		name string
		path string
	}
	providers := []providerInfo{
		{"google_drive", detectGoogleDriveMount()},
		{"onedrive", detectOneDriveFolder()},
		{"dropbox", detectDropboxFolder()},
		{"box", detectBoxFolder()},
		{"pcloud", detectPCloudFolder()},
		{"icloud", detectICloudFolder()},
		{"mega", detectMEGAFolder()},
		{"nextcloud", detectNextcloudFolder()},
		{"syncthing", detectSyncthingFolder()},
	}

	for _, p := range providers {
		if p.path == "" {
			continue
		}
		free := driveFreeGB(p.path)
		sb.WriteString(fmt.Sprintf("  [%-13s] %s  (%.1f GB free)\n", p.name, p.path, free))
		// Mark drive letter as seen to avoid duplicate in drive scan
		if len(p.path) >= 2 && p.path[1] == ':' {
			seen[strings.ToUpper(p.path[:1])] = true
		}
	}

	// Network/SMB shares
	netDrives := detectNetworkDrives()
	for _, nd := range netDrives {
		if !seen[nd.Letter] {
			sb.WriteString(fmt.Sprintf("  [%-13s] %s  (%.1f GB free)\n", "network", nd.Path, nd.FreeGB))
			seen[nd.Letter] = true
		}
	}

	// Remaining drive letters (non-cloud, non-network)
	drives := detectDrives()
	ghAvailable := detectGitHubCLI()

	for _, d := range drives {
		if seen[d.Letter] {
			continue
		}
		if d.IsCloud {
			if d.CloudType == "google_drive" {
				myDrive := filepath.Join(d.Letter+":\\", "My Drive")
				if _, err := os.Stat(myDrive); err == nil {
					sb.WriteString(fmt.Sprintf("  [%-13s] %s  (%.1f GB free)\n", "google_drive", myDrive, d.FreeGB))
					continue
				}
			}
			sb.WriteString(fmt.Sprintf("  [%-13s] %s  (%.1f GB free)\n", d.CloudType, d.Letter+":\\", d.FreeGB))
		} else {
			sb.WriteString(fmt.Sprintf("  [%-13s] %s  (%.1f GB free)\n", "local", d.Letter+":\\", d.FreeGB))
		}
	}

	if ghAvailable {
		sb.WriteString(fmt.Sprintf("  [%-13s] %s\n", "github", "private repo via gh CLI"))
	}

	sb.WriteString("\nTo configure:\n")
	sb.WriteString("  backup_config(provider: \"local\", local_path: \"D:\\\\ai-memory-backups\")\n")
	sb.WriteString("  backup_config(provider: \"google_drive\")\n")
	sb.WriteString("  backup_config(provider: \"onedrive\")\n")
	sb.WriteString("  backup_config(provider: \"dropbox\")\n")
	sb.WriteString("  backup_config(provider: \"box\")\n")
	sb.WriteString("  backup_config(provider: \"pcloud\")\n")
	sb.WriteString("  backup_config(provider: \"icloud\")\n")
	sb.WriteString("  backup_config(provider: \"mega\")\n")
	sb.WriteString("  backup_config(provider: \"nextcloud\")\n")
	sb.WriteString("  backup_config(provider: \"syncthing\")\n")
	sb.WriteString("  backup_config(provider: \"github\")\n")

	return sb.String(), nil
}

func driveFreeGB(path string) float64 {
	fd, err := windowsGetDiskFreeSpace(path)
	if err != nil {
		return 0
	}
	return float64(fd.freeBytes) / (1024 * 1024 * 1024)
}

func handleBackupStatus(store *memory.Store) (interface{}, error) {
	c, err := store.GetBackupConfig()
	if err != nil {
		return nil, err
	}

	cfg, _ := loadBackupConfigFile()
	backups := cfg.Backups

	var sb strings.Builder
	if c == nil {
		sb.WriteString("Backup: NOT CONFIGURED\n\n")
		sb.WriteString("Run list_backup_drives to see available locations.\n")
		return sb.String(), nil
	}

	sb.WriteString("Backup Config:\n")
	sb.WriteString(fmt.Sprintf("  Provider: %s\n", c.Provider))
	if c.LocalPath != "" {
		sb.WriteString(fmt.Sprintf("  Path: %s\n", c.LocalPath))
	}
	sb.WriteString(fmt.Sprintf("  Auto: %v (every %dh)\n", c.AutoBackup, c.IntervalHours))
	if c.LastBackup != "" {
		sb.WriteString(fmt.Sprintf("  Last backup: %s\n", c.LastBackup))
	}
	sb.WriteString(fmt.Sprintf("  Max retained: %d\n", maxBackups))
	sb.WriteString("\n")

	if len(backups) == 0 {
		sb.WriteString("No backups yet.\n")
	} else {
		sb.WriteString(fmt.Sprintf("Backups (%d):\n", len(backups)))
		for i, b := range backups {
			sb.WriteString(fmt.Sprintf("  [%d] %s — %d personas, %d memories, %d skills, %d bytes (%s)\n",
				i+1, b.Timestamp, b.PersonaCount, b.MemoryCount, b.SkillCount, b.FileSize, b.Provider))
		}
	}

	return sb.String(), nil
}

func handleBackup(pm *persona.Manager, store *memory.Store, args map[string]interface{}) (interface{}, error) {
	c, err := store.GetBackupConfig()
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("no backup config. Run backup_config first")
	}

	provider, _ := args["provider"].(string)
	if provider == "" {
		provider = c.Provider
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	archiveName := fmt.Sprintf("ai-memory-backup-%s.zip", timestamp)

	dataDir := dataDir()
	tmpDir := os.TempDir()
	archivePath := filepath.Join(tmpDir, archiveName)

	if err := createBackupArchive(dataDir, archivePath, map[string][]byte{
		"README.md": []byte(generateBackupReadme(pm, store, timestamp, provider)),
	}); err != nil {
		return nil, fmt.Errorf("create archive: %w", err)
	}
	defer os.Remove(archivePath)

	// Real counts from store, not file walking
	personaCount := len(pm.List())
	status, _ := store.Status()
	memoryCount := 0
	skillCount := 0
	if status != nil {
		memoryCount = status.MemoryCount
		skillCount = status.SkillCount
	}

	checksum, fileSize, err := fileChecksum(archivePath)
	if err != nil {
		return nil, fmt.Errorf("checksum: %w", err)
	}

	destPath, err := copyBackupToProvider(provider, archivePath, archiveName, c.LocalPath)
	if err != nil {
		return nil, err
	}

	b, err := store.RecordBackup(timestamp, provider, checksum, destPath, fileSize, personaCount, memoryCount, skillCount)
	if err != nil {
		return nil, err
	}

	store.UpdateBackupLastBackup(c.ID, timestamp)

	// Write to config file + prune old backups
	addBackupRecord(BackupRecord{
		Timestamp:    timestamp,
		Provider:     provider,
		Checksum:     checksum,
		ArchivePath:  destPath,
		FileSize:     fileSize,
		PersonaCount: personaCount,
		MemoryCount:  memoryCount,
		SkillCount:   skillCount,
	})
	pruneOldBackups()

	return fmt.Sprintf("Backup created: %s\nProvider: %s\nChecksum: %s\nSize: %d bytes\nPersonas: %d, Memories: %d, Skills: %d\nDest: %s",
		b.Timestamp, b.Provider, b.Checksum[:16], b.FileSize, b.PersonaCount, b.MemoryCount, b.SkillCount, destPath), nil
}

func handleRestore(pm *persona.Manager, store *memory.Store, args map[string]interface{}) (interface{}, error) {
	backupID, _ := args["backup_id"].(float64)
	provider, _ := args["provider"].(string)

	cfg, err := loadBackupConfigFile()
	if err != nil {
		return nil, fmt.Errorf("load backup config: %w", err)
	}
	if len(cfg.Backups) == 0 {
		return nil, fmt.Errorf("no backups available")
	}

	var target *BackupRecord
	if backupID > 0 {
		for i := range cfg.Backups {
			if int64(i+1) == int64(backupID) {
				target = &cfg.Backups[i]
				break
			}
		}
	} else if provider != "" {
		for i := range cfg.Backups {
			if cfg.Backups[i].Provider == provider {
				target = &cfg.Backups[i]
			}
		}
	} else {
		target = &cfg.Backups[len(cfg.Backups)-1]
	}

	if target == nil {
		return nil, fmt.Errorf("backup not found")
	}

	// Validate checksum + file existence
	if err := validateBackupRecord(target); err != nil {
		return nil, fmt.Errorf("backup validation failed: %w", err)
	}

	archivePath := target.ArchivePath
	if strings.HasPrefix(archivePath, "github://") {
		downloaded, err := downloadFromGitHub(archivePath)
		if err != nil {
			return nil, fmt.Errorf("download from github: %w", err)
		}
		archivePath = downloaded
		defer os.Remove(archivePath)
	}

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("backup file not found: %s", archivePath)
	}

	dataDir := dataDir()
	if err := restoreFromArchive(archivePath, dataDir); err != nil {
		return nil, fmt.Errorf("restore: %w", err)
	}

	return fmt.Sprintf("Restored from backup %s\nProvider: %s\nChecksum: %s\nPersonas: %d, Memories: %d, Skills: %d\nRestart the server to load restored data.",
		target.Timestamp, target.Provider, target.Checksum[:16], target.PersonaCount, target.MemoryCount, target.SkillCount), nil
}

func copyBackupToProvider(provider, archivePath, archiveName, localPath string) (string, error) {
	switch provider {
	case "google_drive", "onedrive", "dropbox", "box", "pcloud", "icloud", "mega", "nextcloud", "syncthing":
		root := findProviderRoot(provider)
		if root == "" {
			return "", fmt.Errorf("%s folder not found — is the client installed?", provider)
		}
		dest := filepath.Join(root, "ai-memory-backups", archiveName)
		return dest, copyFile(archivePath, dest)

	case "github":
		repoName := "ai-memory-backup"
		if err := pushToGitHub(archivePath, archiveName, repoName); err != nil {
			return "", fmt.Errorf("github push: %w", err)
		}
		return fmt.Sprintf("github://%s/%s", repoName, archiveName), nil

	case "local":
		if localPath == "" {
			return "", fmt.Errorf("local_path not set. Run list_backup_drives to see available drives")
		}
		dest := filepath.Join(localPath, archiveName)
		return dest, copyFile(archivePath, dest)

	default:
		return "", fmt.Errorf("unknown provider: %s — use local, google_drive, onedrive, dropbox, box, pcloud, icloud, mega, nextcloud, syncthing, or github", provider)
	}
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func createBackupArchive(dataDir, archivePath string, extraFiles map[string][]byte) error {
	zipFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	err = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		topLevel := strings.SplitN(relPath, string(filepath.Separator), 2)[0]
		if topLevel == "skills" {
			return nil
		}

		f, err := w.Create(relPath)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(f, src)
		return err
	})
	if err != nil {
		return err
	}

	for name, content := range extraFiles {
		f, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := f.Write(content); err != nil {
			return err
		}
	}

	return nil
}

func restoreFromArchive(archivePath, dataDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		outPath := filepath.Join(dataDir, f.Name)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer outFile.Close()
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}
	return nil
}

func fileChecksum(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func pushToGitHub(archivePath, archiveName, repoName string) error {
	cmd := exec.Command("gh", "repo", "create", repoName, "--private", "--description", "ai-memory automated backup")
	cmd.Dir = os.TempDir()
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "already exists") {
			return fmt.Errorf("create repo: %s: %w", string(out), err)
		}
	}

	tmpDir := filepath.Join(os.TempDir(), repoName)
	os.RemoveAll(tmpDir)
	cmd = exec.Command("gh", "repo", "clone", repoName, tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clone: %s: %w", string(out), err)
	}
	defer os.RemoveAll(tmpDir)

	src, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dstPath := filepath.Join(tmpDir, archiveName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	for _, args := range [][]string{
		{"git", "-C", tmpDir, "add", archiveName},
		{"git", "-C", tmpDir, "commit", "-m", fmt.Sprintf("backup: %s", archiveName)},
		{"git", "-C", tmpDir, "push"},
	} {
		cmd = exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s: %w", args[1], string(out), err)
		}
	}

	return nil
}

func downloadFromGitHub(archiveRef string) (string, error) {
	parts := strings.TrimPrefix(archiveRef, "github://")
	slashIdx := strings.Index(parts, "/")
	if slashIdx < 0 {
		return "", fmt.Errorf("invalid github ref: %s", archiveRef)
	}
	repo := parts[:slashIdx]
	file := parts[slashIdx+1:]

	tmpDir := filepath.Join(os.TempDir(), "ai-memory-restore-"+repo)
	os.RemoveAll(tmpDir)

	cmd := exec.Command("gh", "repo", "clone", repo, tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("clone: %s: %w", string(out), err)
	}

	return filepath.Join(tmpDir, file), nil
}

func generateBackupReadme(pm *persona.Manager, store *memory.Store, timestamp, provider string) string {
	var sb strings.Builder

	sb.WriteString("# ai-memory Backup\n\n")
	sb.WriteString(fmt.Sprintf("- **Timestamp**: %s\n", timestamp))
	sb.WriteString(fmt.Sprintf("- **Provider**: %s\n", provider))
	sb.WriteString(fmt.Sprintf("- **Generated by**: ai-memory %s\n\n", version.Full()))

	personas := pm.List()
	sb.WriteString(fmt.Sprintf("## Personas (%d)\n\n", len(personas)))
	for _, p := range personas {
		sb.WriteString(fmt.Sprintf("### %s\n", p.Name))
		if p.Identity != "" {
			sb.WriteString(fmt.Sprintf("- **Identity**: %s\n", p.Identity))
		}
		if p.Description != "" {
			sb.WriteString(fmt.Sprintf("- **Description**: %s\n", p.Description))
		}
		if p.Tone != "" {
			sb.WriteString(fmt.Sprintf("- **Tone**: %s\n", p.Tone))
		}
		if p.Greeting != "" {
			sb.WriteString(fmt.Sprintf("- **Greeting**: %s\n", p.Greeting))
		}
		if len(p.Skills) > 0 {
			sb.WriteString(fmt.Sprintf("- **Skills**: %s\n", strings.Join(p.Skills, ", ")))
		}
		sb.WriteString("\n")
	}

	if shared, err := store.ListAll(); err == nil && len(shared) > 0 {
		sb.WriteString(fmt.Sprintf("## Shared Memories (%d)\n\n", len(shared)))
		for _, m := range shared {
			sb.WriteString(fmt.Sprintf("- **[%s]** %s → %s\n", m.Impact, truncate(m.Experience, 80), truncate(m.Lesson, 80)))
		}
		sb.WriteString("\n")
	}

	if profile, err := store.ListUserProfile(); err == nil && len(profile) > 0 {
		sb.WriteString("## User Profile\n\n")
		for _, u := range profile {
			sb.WriteString(fmt.Sprintf("- **%s**: %s (confidence: %.0f%%)\n", u.Field, u.Value, u.Confidence*100))
		}
		sb.WriteString("\n")
	}

	if projects, err := store.ListProjectContexts(); err == nil && len(projects) > 0 {
		sb.WriteString("## Project Contexts\n\n")
		for _, p := range projects {
			active := ""
			if p.IsActive {
				active = " *(active)*"
			}
			sb.WriteString(fmt.Sprintf("- **%s**%s — %s (%s)\n", p.Name, active, p.Root, p.Type))
		}
		sb.WriteString("\n")
	}

	if mappings, err := store.ListPersonaMappings(); err == nil && len(mappings) > 0 {
		sb.WriteString("## Persona Mappings\n\n")
		sb.WriteString("| Project | Persona |\n")
		sb.WriteString("|---------|---------|\n")
		for _, m := range mappings {
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", m.Project, m.Persona))
		}
		sb.WriteString("\n")
	}

	if usage, err := store.ListSkillUsage(10); err == nil && len(usage) > 0 {
		sb.WriteString("## Recent Skill Usage\n\n")
		for _, u := range usage {
			sb.WriteString(fmt.Sprintf("- **%s** — %s [%s]\n", u.Skill, u.Context, u.Outcome))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("Restore this backup with: `restore(backup_id: <id>)` or `restore(provider: \"\")`\n")

	return sb.String()
}

// startAutoBackup runs a background ticker that triggers backups at the configured interval.
func startAutoBackup(pm *persona.Manager, store *memory.Store) {
	for {
		c, err := store.GetBackupConfig()
		if err != nil || c == nil || !c.AutoBackup || c.IntervalHours <= 0 {
			time.Sleep(5 * time.Minute)
			continue
		}

		interval := time.Duration(c.IntervalHours) * time.Hour
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			// Re-read config each tick in case it changed
			cfg, err := store.GetBackupConfig()
			if err != nil || cfg == nil || !cfg.AutoBackup || cfg.IntervalHours <= 0 {
				break
			}

			provider := cfg.Provider
			timestamp := time.Now().UTC().Format("20060102-150405")
			archiveName := fmt.Sprintf("ai-memory-backup-%s.zip", timestamp)

			dataDir := dataDir()
			tmpDir := os.TempDir()
			archivePath := filepath.Join(tmpDir, archiveName)

			if err := createBackupArchive(dataDir, archivePath, map[string][]byte{
				"README.md": []byte(generateBackupReadme(pm, store, timestamp, provider)),
			}); err != nil {
				fmt.Fprintf(os.Stderr, "auto-backup: create archive: %v\n", err)
				continue
			}

			checksum, fileSize, err := fileChecksum(archivePath)
			if err != nil {
				os.Remove(archivePath)
				fmt.Fprintf(os.Stderr, "auto-backup: checksum: %v\n", err)
				continue
			}

			destPath, err := copyBackupToProvider(provider, archivePath, archiveName, cfg.LocalPath)
			os.Remove(archivePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "auto-backup: copy to %s: %v\n", provider, err)
				continue
			}

			personaCount := len(pm.List())
			status, _ := store.Status()
			memoryCount, skillCount := 0, 0
			if status != nil {
				memoryCount = status.MemoryCount
				skillCount = status.SkillCount
			}

			store.RecordBackup(timestamp, provider, checksum, destPath, fileSize, personaCount, memoryCount, skillCount)
			store.UpdateBackupLastBackup(cfg.ID, timestamp)
		}
	}
}
