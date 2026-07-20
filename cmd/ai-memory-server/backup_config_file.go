package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const maxBackups = 3

type BackupRecord struct {
	Timestamp    string `json:"timestamp"`
	Provider     string `json:"provider"`
	Checksum     string `json:"checksum"`
	ArchivePath  string `json:"archive_path"`
	FileSize     int64  `json:"file_size"`
	PersonaCount int    `json:"persona_count"`
	MemoryCount  int    `json:"memory_count"`
	SkillCount   int    `json:"skill_count"`
}

type BackupConfigFile struct {
	Backups []BackupRecord `json:"backups"`
}

func backupConfigPath() string {
	return filepath.Join(dataDir(), "backups.json")
}

func loadBackupConfigFile() (*BackupConfigFile, error) {
	path := backupConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &BackupConfigFile{}, nil
		}
		return nil, err
	}
	var cfg BackupConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveBackupConfigFile(cfg *BackupConfigFile) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(backupConfigPath(), data, 0o644)
}

func addBackupRecord(rec BackupRecord) error {
	cfg, err := loadBackupConfigFile()
	if err != nil {
		return err
	}
	cfg.Backups = append(cfg.Backups, rec)
	return saveBackupConfigFile(cfg)
}

func getLatestBackup() (*BackupRecord, error) {
	cfg, err := loadBackupConfigFile()
	if err != nil {
		return nil, err
	}
	if len(cfg.Backups) == 0 {
		return nil, fmt.Errorf("no backups in config")
	}
	return &cfg.Backups[len(cfg.Backups)-1], nil
}

func findBackupByID(id int64) (*BackupRecord, error) {
	cfg, err := loadBackupConfigFile()
	if err != nil {
		return nil, err
	}
	for i := range cfg.Backups {
		// Use index+1 as ID (1-based)
		if int64(i+1) == id {
			return &cfg.Backups[i], nil
		}
	}
	return nil, fmt.Errorf("backup %d not found in config", id)
}

func pruneOldBackups() error {
	cfg, err := loadBackupConfigFile()
	if err != nil {
		return err
	}
	if len(cfg.Backups) <= maxBackups {
		return nil
	}

	sort.Slice(cfg.Backups, func(i, j int) bool {
		return cfg.Backups[i].Timestamp < cfg.Backups[j].Timestamp
	})

	toDelete := cfg.Backups[:len(cfg.Backups)-maxBackups]
	kept := cfg.Backups[len(cfg.Backups)-maxBackups:]

	for _, rec := range toDelete {
		deleteBackupFile(rec)
	}

	cfg.Backups = kept
	return saveBackupConfigFile(cfg)
}

func deleteBackupFile(rec BackupRecord) {
	if rec.Provider == "github" {
		deleteFromGitHub(rec.ArchivePath)
		return
	}
	os.Remove(rec.ArchivePath)
}

func deleteFromGitHub(archiveRef string) {
	parts := strings.TrimPrefix(archiveRef, "github://")
	slashIdx := strings.Index(parts, "/")
	if slashIdx < 0 {
		return
	}
	repo := parts[:slashIdx]
	file := parts[slashIdx+1:]

	tmpDir := filepath.Join(os.TempDir(), "ai-memory-prune-"+repo)
	os.RemoveAll(tmpDir)

	if err := exec.Command("gh", "repo", "clone", repo, tmpDir).Run(); err != nil {
		return
	}
	defer os.RemoveAll(tmpDir)

	os.Remove(filepath.Join(tmpDir, file))

	exec.Command("git", "-C", tmpDir, "add", "-A").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", fmt.Sprintf("prune: remove %s", file)).Run()
	exec.Command("git", "-C", tmpDir, "push").Run()
}

func validateBackupRecord(rec *BackupRecord) error {
	switch {
	case strings.HasPrefix(rec.ArchivePath, "github://"):
		parts := strings.TrimPrefix(rec.ArchivePath, "github://")
		slashIdx := strings.Index(parts, "/")
		if slashIdx < 0 {
			return fmt.Errorf("invalid github ref: %s", rec.ArchivePath)
		}
		repo := parts[:slashIdx]
		out, err := exec.Command("gh", "api", fmt.Sprintf("repos/%s", repo), "--jq", ".name").Output()
		if err != nil || strings.TrimSpace(string(out)) == "" {
			return fmt.Errorf("github repo %s not accessible", repo)
		}
		return nil
	default:
		if _, err := os.Stat(rec.ArchivePath); err != nil {
			return fmt.Errorf("backup file missing: %s", rec.ArchivePath)
		}
		f, err := os.Open(rec.ArchivePath)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		actual := hex.EncodeToString(h.Sum(nil))
		if actual != rec.Checksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", rec.Checksum[:16], actual[:16])
		}
		return nil
	}
}
