package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

var (
	// runWorktree specifies which worktree to execute the command in (optional)
	runWorktree string
)

// isSubDir checks if child is a subdirectory of parent (or the same directory)
func isSubDir(parent, child string) bool {
	// Resolve symlinks
	absParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		absParent = parent
	}

	absChild, err := filepath.EvalSymlinks(child)
	if err != nil {
		absChild = child
	}

	rel, err := filepath.Rel(absParent, absChild)
	if err != nil {
		return false
	}

	// rel == "." -> same dir
	// rel starts with ".." -> outside
	return rel == "." || (!strings.HasPrefix(rel, "..") && !strings.HasPrefix(rel, "/"))
}

// determineExecDir determines the execution directory and target worktree name
// based on current working directory and user input
func determineExecDir(wm *workspace.WorkspaceManager, cwd string, targetWorktree string) (string, string, error) {
	// If user specified a worktree context
	if targetWorktree != "" {
		node, exists := wm.State.Nodes[targetWorktree]
		if !exists {
			return "", "", fmt.Errorf("node '%s' does not exist", targetWorktree)
		}

		// If we are already inside the target worktree (e.g. sub-directory), stay there
		if isSubDir(node.WorktreePath, cwd) {
			return cwd, targetWorktree, nil
		}
		// Otherwise switch to the root of the worktree
		return node.WorktreePath, targetWorktree, nil
	}

	// Default to Main Repo Root (No -w specified)
	// ALWAYS run in main_repo root, regardless of current directory.
	return wm.State.RepoPath, "", nil
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Execute commands in the main_repo or specified worktree context",
	Long: `Execute arbitrary commands in the main_repo or specified worktree context.

By default, commands are executed in the main_repo directory.
Use the --worktree (-w) flag to specify a node's worktree for execution.

Examples:
  # Execute git pull in the main repository
  orion run git pull

  # Execute git fetch origin in the main repository
  orion run git fetch origin

  # Execute make build in the main repository
  orion run make build

  # Execute commands in a specific node's worktree
  orion run -w my-node npm test
  orion run --worktree my-node go test ./...`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Error: A command must be specified")
			fmt.Println("Usage: orion run <command> [args...]")
			os.Exit(1)
		}

		// 解析 flags：找到 --worktree 或 -w 及其参数
		// 剩下的 args 就是要执行的命令
		var commandArgs []string
		targetWorktree := ""
		i := 0
		for i < len(args) {
			arg := args[i]
			if arg == "--worktree" || arg == "-w" {
				if i+1 >= len(args) {
					fmt.Println("Error: --worktree/-w requires an argument")
					os.Exit(1)
				}
				targetWorktree = args[i+1]
				i += 2
				continue
			}
			// 遇到非 flag 参数，之后的所有内容都是命令参数
			commandArgs = args[i:]
			break
		}

		if len(commandArgs) == 0 {
			fmt.Println("Error: A command must be specified")
			fmt.Println("Usage: orion run <command> [args...]")
			os.Exit(1)
		}

		// 找到 workspace root
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error: Failed to get current directory: %v\n", err)
			os.Exit(1)
		}

		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			fmt.Printf("Error: Failed to load workspace: %v\n", err)
			os.Exit(1)
		}

		// 确定执行目录
		var execDir string
		var worktreeName string
		execDir, worktreeName, err = determineExecDir(wm, cwd, targetWorktree)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Verify directory exists
		if _, err := os.Stat(execDir); os.IsNotExist(err) {
			fmt.Printf("Error: Execution directory does not exist: %s\n", execDir)
			os.Exit(1)
		}

		// 执行命令
		command := exec.Command(commandArgs[0], commandArgs[1:]...)
		command.Dir = execDir
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		// 设置环境变量，标记当前在 orion run 上下文中
		command.Env = os.Environ()
		command.Env = append(command.Env, "ORION_RUN=1")
		if worktreeName != "" {
			command.Env = append(command.Env, fmt.Sprintf("ORION_WORKTREE=%s", worktreeName))
		}

		err = command.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Command returned non-zero exit code
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			}
			fmt.Fprintf(os.Stderr, "Error: Failed to execute command: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Note: Since DisableFlagParsing=true, this flag definition is only for help documentation generation
	// Actual parsing is handled manually in the Run function
	runCmd.Flags().StringVarP(&runWorktree, "worktree", "w", "", "Specify which node's worktree to execute the command in")
}

// GetRunWorktreePath 是一个辅助函数，用于获取 run 命令的执行目录
// 主要用于测试
func GetRunWorktreePath(rootPath, worktreeName string) (string, error) {
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		return "", err
	}

	if worktreeName == "" {
		return wm.State.RepoPath, nil
	}

	node, exists := wm.State.Nodes[worktreeName]
	if !exists {
		return "", fmt.Errorf("node '%s' does not exist", worktreeName)
	}

	return node.WorktreePath, nil
}

// ExecuteInWorktree 在指定 worktree 中执行命令，返回输出
// 主要用于测试
func ExecuteInWorktree(rootPath, worktreeName string, args []string) (string, int, error) {
	var execDir string
	var err error

	if worktreeName == "" {
		execDir, err = GetRunWorktreePath(rootPath, "")
		if err != nil {
			return "", -1, err
		}
	} else {
		execDir, err = GetRunWorktreePath(rootPath, worktreeName)
		if err != nil {
			return "", -1, err
		}
	}

	if len(args) == 0 {
		return "", -1, fmt.Errorf("no command specified")
	}

	command := exec.Command(args[0], args[1:]...)
	command.Dir = execDir

	output, err := command.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			return "", -1, err
		}
	}

	return string(output), exitCode, nil
}
