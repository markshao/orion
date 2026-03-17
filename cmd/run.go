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
	// runWorktree 指定要在哪个 worktree 中执行命令（可选）
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
	Short: "在 main_repo 或指定 worktree 上下文中执行命令",
	Long: `在 main_repo 或指定 worktree 上下文中执行任意命令。

默认情况下，命令会在 main_repo 目录下执行。
使用 --worktree (-w) 标志可以指定在某个 node 的 worktree 中执行。

示例:
  # 在主仓库执行 git pull
  orion run git pull

  # 在主仓库执行 git fetch origin
  orion run git fetch origin

  # 在主仓库执行 make build
  orion run make build

  # 在指定 node 的 worktree 中执行命令
  orion run -w my-node npm test
  orion run --worktree my-node go test ./...`,
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Error: 需要指定要执行的命令")
			fmt.Println("用法: orion run <command> [args...]")
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
					fmt.Println("Error: --worktree/-w 需要一个参数")
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
			fmt.Println("Error: 需要指定要执行的命令")
			fmt.Println("用法: orion run <command> [args...]")
			os.Exit(1)
		}

		// 找到 workspace root
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error: 获取当前目录失败: %v\n", err)
			os.Exit(1)
		}

		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			fmt.Printf("Error: 加载 workspace 失败: %v\n", err)
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

		// 验证目录存在
		if _, err := os.Stat(execDir); os.IsNotExist(err) {
			fmt.Printf("Error: 执行目录不存在: %s\n", execDir)
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
				// 命令返回非零退出码
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			}
			fmt.Fprintf(os.Stderr, "Error: 执行命令失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// 注意：由于 DisableFlagParsing=true，这里的 flag 定义仅用于帮助文档生成
	// 实际解析在 Run 函数中手动处理
	runCmd.Flags().StringVarP(&runWorktree, "worktree", "w", "", "指定要在哪个 node 的 worktree 中执行命令")
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
