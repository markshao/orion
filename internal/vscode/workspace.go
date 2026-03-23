package vscode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WorkspaceFile struct {
	Folders  []Folder               `json:"folders"`
	Settings map[string]interface{} `json:"settings"`
}

type Folder struct {
	Path string `json:"path"`
}

// UpdateWorkspaceFile updates or creates the .code-workspace file
func UpdateWorkspaceFile(rootPath string, repoDir string, nodesDir string, nodes []string) error {
	projectName := filepath.Base(rootPath)
	// Remove _swarm suffix if present for cleaner name
	projectName = strings.TrimSuffix(projectName, "_swarm")

	workspaceFilePath := filepath.Join(rootPath, fmt.Sprintf("%s.code-workspace", projectName))

	var folders []Folder
	if strings.TrimSpace(repoDir) != "" {
		folders = append(folders, Folder{Path: repoDir})
	}

	for _, node := range nodes {
		folders = append(folders, Folder{
			Path: filepath.Join(nodesDir, node),
		})
	}

	workspace := WorkspaceFile{
		Folders:  folders,
		Settings: map[string]interface{}{},
	}

	file, err := os.Create(workspaceFilePath)
	if err != nil {
		return fmt.Errorf("failed to create workspace file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(workspace); err != nil {
		return fmt.Errorf("failed to encode workspace file: %w", err)
	}

	return nil
}
