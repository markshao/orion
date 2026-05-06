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
	Label      string `json:"label"`       // 一句话任务摘要 (same language as input)
}

// Client 是 langchaingo 封装的 AI 客户端
type Client struct {
	llm   llms.Model
	model string
}

// NewClient 创建 AI 客户端，从 ~/.orion.yaml 读取配置
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
	// 构建 system prompt
	systemPrompt := `你是一个 Git 分支命名专家。根据用户的开发任务描述，生成合适的分支名、node 名和任务摘要 label。

命名规则:
- 分支名格式: feature/xxx, fix/xxx, chore/xxx (kebab-case)
- Node 名格式: 简短可读，kebab-case，建议以 -dev 或 -fix 结尾
- 新功能用 feature/ 前缀
- Bug 修复用 fix/ 前缀  
- 重构/优化用 chore/ 前缀
- 默认基于 main 分支，除非用户明确指定其他分支

label 规则:
- 用和用户描述相同的语言
- 1 行，尽量简短
- 描述“要做什么”，不要写步骤，不要加引号

你必须以 JSON 格式输出，不要包含任何其他解释或 markdown 标记:
{
  "branch_name": "分支名",
  "node_name": "node 名",
  "base_branch": "基础分支名",
  "label": "一句话任务摘要"
}`

	// 构建用户 prompt
	userPrompt := fmt.Sprintf(`开发任务描述: "%s"

请分析上述描述，生成合适的分支名、node 名、基础分支名和一句话任务摘要 label。`, description)

	content, err := c.GenerateText(systemPrompt, userPrompt, 0.2, 256)
	if err != nil {
		return nil, err
	}

	// 解析 JSON 响应
	return parseSpawnPlan(content)
}

// GenerateText sends a system and user prompt and returns the first text choice.
func (c *Client) GenerateText(systemPrompt, userPrompt string, temperature float64, maxTokens int) (string, error) {
	ctx := context.Background()

	// 调用 LLM
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	effectiveTemp := normalizeTemperatureForModel(c.model, temperature)
	response, err := c.llm.GenerateContent(ctx, messages,
		llms.WithTemperature(effectiveTemp),
		llms.WithMaxTokens(maxTokens),
	)
	if err != nil {
		// Some models (e.g. OpenAI reasoning models like o1/o3) only accept temperature=1.
		// If we hit that constraint, retry once with temperature=1 to avoid breaking users
		// who set these models in ~/.orion.yaml.
		if effectiveTemp != 1 && shouldForceTemperatureOne(err) {
			retryResp, retryErr := c.llm.GenerateContent(ctx, messages,
				llms.WithTemperature(1),
				llms.WithMaxTokens(maxTokens),
			)
			if retryErr == nil {
				response = retryResp
			} else {
				return "", fmt.Errorf("LLM call failed (temp=%.2f, retry temp=1): %w", effectiveTemp, retryErr)
			}
		} else {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}
	return response.Choices[0].Content, nil
}

func normalizeTemperatureForModel(model string, requested float64) float64 {
	m := strings.ToLower(strings.TrimSpace(model))
	// Heuristic: OpenAI reasoning models (o1/o3/...) require temperature=1.
	if len(m) >= 2 && m[0] == 'o' && m[1] >= '0' && m[1] <= '9' {
		return 1
	}
	return requested
}

func shouldForceTemperatureOne(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "invalid temperature") && strings.Contains(s, "only 1")
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
		plan.BaseBranch = "origin/main"
	}

	plan.Label = strings.TrimSpace(plan.Label)
	plan.Label = strings.ReplaceAll(plan.Label, "\n", " ")
	plan.Label = strings.ReplaceAll(plan.Label, "\r", " ")

	return &plan, nil
}
