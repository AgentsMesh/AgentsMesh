package extension

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// --- Repo type detection ---

// detectRepoType determines if a repo is a single-skill or collection
func detectRepoType(repoDir string) string {
	// Check root for SKILL.md -> single
	if fileExists(filepath.Join(repoDir, "SKILL.md")) {
		return "single"
	}
	return "collection"
}

// --- Skill scanning ---

// scanCollectionSkills scans a collection repo for skill directories
func scanCollectionSkills(repoDir string) ([]SkillInfo, error) {
	var skills []SkillInfo

	// Priority 1: Check skills/ subdirectory
	skillsDir := filepath.Join(repoDir, "skills")
	if dirExists(skillsDir) {
		entries, err := os.ReadDir(skillsDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() || shouldIgnoreDir(entry.Name()) {
					continue
				}
				dirPath := filepath.Join(skillsDir, entry.Name())
				if fileExists(filepath.Join(dirPath, "SKILL.md")) {
					info, err := parseSkillDir(dirPath)
					if err != nil {
						slog.Warn("Failed to parse skill", "dir", dirPath, "error", err)
						continue
					}
					skills = append(skills, *info)
				}
			}
		}
		if len(skills) > 0 {
			return skills, nil
		}
	}

	// Priority 2: Scan root-level subdirectories
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || shouldIgnoreDir(entry.Name()) {
			continue
		}
		dirPath := filepath.Join(repoDir, entry.Name())
		if fileExists(filepath.Join(dirPath, "SKILL.md")) {
			info, err := parseSkillDir(dirPath)
			if err != nil {
				slog.Warn("Failed to parse skill", "dir", dirPath, "error", err)
				continue
			}
			skills = append(skills, *info)
		}
	}

	return skills, nil
}

// ignoredDirs is the set of directory names to skip during skill scanning.
// Extracted as a package-level variable to avoid re-creating the map on every call.
var ignoredDirs = map[string]bool{
	".git": true, ".github": true, ".vscode": true,
	"spec": true, "template": true, "templates": true, ".claude-plugin": true,
	"node_modules": true, "__pycache__": true, "vendor": true,
}

// shouldIgnoreDir returns true for directories to skip during scanning
func shouldIgnoreDir(name string) bool {
	return strings.HasPrefix(name, ".") || ignoredDirs[name]
}

// parseSkillDir parses a skill directory's SKILL.md frontmatter
func parseSkillDir(dirPath string) (*SkillInfo, error) {
	skillMdPath := filepath.Join(dirPath, "SKILL.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	fm := parseFrontmatter(string(content))

	slug := fm["name"]
	if slug == "" {
		// Fallback to directory name
		slug = filepath.Base(dirPath)
	}

	return &SkillInfo{
		Slug:          slug,
		DisplayName:   fm["name"],
		Description:   fm["description"],
		License:       fm["license"],
		Compatibility: fm["compatibility"],
		AllowedTools:  fm["allowed-tools"],
		Category:      fm["category"],
		DirPath:       dirPath,
	}, nil
}

// parseFrontmatter extracts YAML-like frontmatter from a markdown file
// Supports simple key: value pairs between --- delimiters
func parseFrontmatter(content string) map[string]string {
	fm := make(map[string]string)

	lines := strings.Split(content, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return fm
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			break
		}
		// Simple key: value parsing
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove surrounding quotes
			value = strings.Trim(value, `"'`)
			fm[key] = value
		}
	}

	return fm
}

// --- File helpers ---

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
