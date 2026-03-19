package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// SpawnPlan 表示 AI 生成的 spawn 计划
type SpawnPlan struct {
	BranchName string `json:"branch_name"` // 如: feature/user-login
	NodeName   string `json:"node_name"`   // 如: user-login-dev
	BaseBranch string `json:"base_branch"` // 如: main, release/v1.2
}

// Client 是 langchaingo 封装的 AI 客户端
type Client struct {
	llm   llms.Model
	model string
}

// NewClient 创建 AI 客户端，从 ~/.orion.conf 读取配置
func NewClient() (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// 使用 OpenAI 兼容接口
	llm, err := openai.New(
		openai.WithToken(cfg.APIKey),
		openai.WithBaseURL(cfg.BaseURL),
		openai.WithModel(cfg.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &Client{
		llm:   llm,
		model: cfg.Model,
	}, nil
}

// GenerateSpawnPlan 根据用户描述生成分支和 node 名称
func (c *Client) GenerateSpawnPlan(description string) (*SpawnPlan, error) {
	ctx := context.Background()

	// 构建 system prompt
	systemPrompt := `你是一个 Git 分支命名专家。根据用户的开发任务描述，生成合适的分支名和 node 名。

命名规则:
- 分支名格式: feature/xxx, fix/xxx, chore/xxx (kebab-case)
- Node 名格式: 简短可读，kebab-case，建议以 -dev 或 -fix 结尾
- 新功能用 feature/ 前缀
- Bug 修复用 fix/ 前缀  
- 重构/优化用 chore/ 前缀
- 默认基于 main 分支，除非用户明确指定其他分支

你必须以 JSON 格式输出，不要包含任何其他解释或 markdown 标记:
{
  "branch_name": "分支名",
  "node_name": "node 名",
  "base_branch": "基础分支名"
}`

	// 构建用户 prompt
	userPrompt := fmt.Sprintf(`开发任务描述: "%s"

请分析上述描述，生成合适的分支名、node 名和基础分支名。`, description)

	// 调用 LLM
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	response, err := c.llm.GenerateContent(ctx, messages,
		llms.WithTemperature(0.2),
		llms.WithMaxTokens(256),
	)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// 解析 JSON 响应
	content := response.Choices[0].Content
	return parseSpawnPlan(content)
}

// parseSpawnPlan 解析 LLM 返回的 JSON
func parseSpawnPlan(content string) (*SpawnPlan, error) {
	// 清理可能的 markdown 代码块
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var plan SpawnPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w\nContent: %s", err, content)
	}

	// 验证必要字段
	if plan.BranchName == "" || plan.NodeName == "" {
		return nil, fmt.Errorf("invalid plan: branch_name or node_name is empty")
	}

	// 设置默认值
	if plan.BaseBranch == "" {
		plan.BaseBranch = "main"
	}

	return &plan, nil
}
