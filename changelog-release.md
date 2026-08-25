## [0.1.6] - 2026-07-20

### Added
- Backup/restore system with 11 providers: local, Google Drive, OneDrive, Dropbox, Box, pCloud, iCloud, MEGA, Nextcloud, Syncthing, GitHub (private repos via `gh` CLI)
- `backup_config`, `backup`, `restore`, `backup_status`, `list_backup_drives` tools
- `backup://status` resource
- Cloud drive detection scans all 26 drive letters for filesystem markers
- Google Drive detected via drive letter `My Drive` subfolder, home folder mount, or registry admin policy mount points
- OneDrive detected via `OneDrive` env var and registry `UserFolder`
- SMB/network share detection via `GetVolumeInformationW` UNC prefix check
- Persona mapping: `map_persona`, `unmap_persona`, `list_persona_mappings` tools
- README.md inside backup archives with persona identities, shared memories, user profile, project contexts, persona mappings, and recent skill usage
- `backups.json` tracks all backups with SHA-256 checksums for restore validation
- Backup pruning: keeps last 3 backups, deletes older from disk or GitHub repo
- Restore validates checksum before extracting
- Auto-backup goroutine runs in background when enabled

### Changed
- `backup_config` accepts `provider`, `local_path`, `auto_backup`, `interval_hours` only

### Fixed
- `pm.Switch()` return value not handled in project handlers
