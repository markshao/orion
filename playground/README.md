# Playground

这个目录用于进行 DevSwarm 的**端到端 (End-to-End) 测试**和集成调试。

## 目录结构

- `test_e2e.sh`: 自动化的 E2E 测试脚本。
- `DevSwarm_workspace/`: 运行测试脚本时生成的临时工作区（已被 gitignore）。

## 如何运行端到端测试

我们提供了一个全自动的测试脚本 `test_e2e.sh`，它会模拟完整的用户工作流：

1.  **自动编译**：重新编译并安装最新的 `devswarm` 二进制文件。
2.  **环境初始化**：拉取远程仓库 (`markshao/DevSwarm`) 并初始化 Workspace。
3.  **创建 Node**：基于 `main` 分支创建一个新的开发节点 (`test-node-1`)。
4.  **模拟开发**：在 Node 中创建文件并提交代码。
5.  **合并验证**：将 Node 的更改 Squash Merge 回主分支，并验证文件是否存在。
6.  **清理**：测试完成后可以选择手动删除生成的 `DevSwarm_workspace`。

### 运行命令

在项目根目录下执行：

```bash
./playground/test_e2e.sh
```

或者进入 playground 目录执行：

```bash
cd playground
./test_e2e.sh
```

### 预期输出

如果一切正常，您将看到类似以下的输出：

```text
🐝 Starting End-to-End Test for DevSwarm...
📦 Rebuilding and installing DevSwarm...
...
🚀 Initializing workspace for https://github.com/markshao/DevSwarm.git...
...
🌱 Spawning node 'test-node-1' on branch 'feature/e2e-test'...
...
✍️ Simulating work in node...
💾 Committing changes in shadow branch...
🔀 Merging node back to logical branch...
...
🎉 E2E Test Completed Successfully!
```

## 注意事项

- 该测试脚本会**删除** `playground/DevSwarm_workspace` 目录下的所有内容，请勿在该目录存放重要文件。
- 测试过程中会产生真实的 git 操作（clone, commit, merge），但都是在本地的临时仓库中进行，不会影响远程仓库。
