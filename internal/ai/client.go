package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"orion/internal/log"
	"strings"
	"time"

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
	log.Info("ai: GenerateSpawnPlan start description=%q model=%q", description, c.model)

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
		log.Error("ai: GenerateSpawnPlan initial request failed description=%q err=%v", description, err)
		return nil, err
	}

	// 解析 JSON 响应
	plan, parseErr := parseSpawnPlan(content)
	if parseErr == nil {
		log.Info("ai: GenerateSpawnPlan parsed plan branch=%q node=%q base=%q label=%q", plan.BranchName, plan.NodeName, plan.BaseBranch, plan.Label)
		return plan, nil
	}
	log.Error("ai: GenerateSpawnPlan parse failed description=%q err=%v raw_response=%q", description, parseErr, content)

	// Some models occasionally ignore the JSON-only instruction on the first try.
	// Retry once with a stricter prompt before surfacing the parsing error.
	repairPrompt := fmt.Sprintf(`开发任务描述: %q

请重新生成结果，并严格遵守以下要求:
- 只输出一个合法 JSON 对象
- 不要包含 markdown、解释、Usage、Flags 或其他文本
- 字段必须是 branch_name、node_name、base_branch、label
- branch_name 使用 feature/xxx、fix/xxx、chore/xxx 之一
- node_name 使用 kebab-case
- label 使用和用户描述相同的语言
`, description)

	repairedContent, repairErr := c.GenerateText(systemPrompt, repairPrompt, 0.2, 256)
	if repairErr == nil {
		if repairedPlan, repairedParseErr := parseSpawnPlan(repairedContent); repairedParseErr == nil {
			log.Info("ai: GenerateSpawnPlan repaired plan branch=%q node=%q base=%q label=%q", repairedPlan.BranchName, repairedPlan.NodeName, repairedPlan.BaseBranch, repairedPlan.Label)
			return repairedPlan, nil
		} else {
			log.Error("ai: GenerateSpawnPlan repaired parse failed description=%q err=%v raw_response=%q", description, repairedParseErr, repairedContent)
		}
	} else {
		log.Error("ai: GenerateSpawnPlan repair request failed description=%q err=%v", description, repairErr)
	}

	return nil, parseErr
}

// GenerateText sends a system and user prompt and returns the first text choice.
func (c *Client) GenerateText(systemPrompt, userPrompt string, temperature float64, maxTokens int) (string, error) {
	ctx := context.Background()
	logLLMRequest(c.model, systemPrompt, userPrompt, temperature, maxTokens)

	// 调用 LLM
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	effectiveTemp := normalizeTemperatureForModel(c.model, temperature)
	response, err := c.generateContentWithRetry(ctx, messages, effectiveTemp, maxTokens)
	if err != nil {
		log.Error("ai: GenerateText failed model=%q requested_temp=%.2f effective_temp=%.2f max_tokens=%d err=%v", c.model, temperature, effectiveTemp, maxTokens, err)
		return "", err
	}

	if len(response.Choices) == 0 {
		log.Error("ai: GenerateText empty response model=%q requested_temp=%.2f effective_temp=%.2f max_tokens=%d", c.model, temperature, effectiveTemp, maxTokens)
		return "", fmt.Errorf("no response from LLM")
	}
	logLLMResponse(c.model, response.Choices[0].Content)
	return response.Choices[0].Content, nil
}

func (c *Client) generateContentWithRetry(ctx context.Context, messages []llms.MessageContent, temperature float64, maxTokens int) (*llms.ContentResponse, error) {
	resp, err := c.generateContentAttempts(ctx, messages, temperature, maxTokens)
	if err == nil {
		return resp, nil
	}

	// Some models (e.g. OpenAI reasoning models like o1/o3) only accept temperature=1.
	if temperature != 1 && shouldForceTemperatureOne(err) {
		retryResp, retryErr := c.generateContentAttempts(ctx, messages, 1, maxTokens)
		if retryErr == nil {
			return retryResp, nil
		}
		return nil, fmt.Errorf("LLM call failed (temp=%.2f, retry temp=1): %w", temperature, retryErr)
	}

	return nil, fmt.Errorf("LLM call failed: %w", err)
}

func (c *Client) generateContentAttempts(ctx context.Context, messages []llms.MessageContent, temperature float64, maxTokens int) (*llms.ContentResponse, error) {
	log.Info("ai: GenerateContent attempt=1 model=%q temperature=%.2f max_tokens=%d", c.model, temperature, maxTokens)
	resp, err := c.llm.GenerateContent(ctx, messages,
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(maxTokens),
	)
	if err == nil {
		log.Info("ai: GenerateContent attempt=1 succeeded model=%q choices=%d", c.model, len(resp.Choices))
		return resp, nil
	}
	log.Error("ai: GenerateContent attempt=1 failed model=%q temperature=%.2f max_tokens=%d err=%v", c.model, temperature, maxTokens, err)
	if !shouldRetryLLMError(err) {
		return nil, err
	}

	lastErr := err
	for attempt := 1; attempt <= 2; attempt++ {
		time.Sleep(time.Duration(attempt) * time.Second)
		log.Info("ai: GenerateContent retry=%d model=%q temperature=%.2f max_tokens=%d", attempt, c.model, temperature, maxTokens)
		retryResp, retryErr := c.llm.GenerateContent(ctx, messages,
			llms.WithTemperature(temperature),
			llms.WithMaxTokens(maxTokens),
		)
		if retryErr == nil {
			log.Info("ai: GenerateContent retry=%d succeeded model=%q choices=%d", attempt, c.model, len(retryResp.Choices))
			return retryResp, nil
		}
		log.Error("ai: GenerateContent retry=%d failed model=%q temperature=%.2f max_tokens=%d err=%v", attempt, c.model, temperature, maxTokens, retryErr)
		lastErr = retryErr
		if !shouldRetryLLMError(retryErr) {
			break
		}
	}

	return nil, fmt.Errorf("LLM call failed after retries: %w", lastErr)
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

func shouldRetryLLMError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "429") ||
		strings.Contains(s, "rate limit") ||
		strings.Contains(s, "overloaded") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "temporarily unavailable")
}

// parseSpawnPlan 解析 LLM 返回的 JSON
func parseSpawnPlan(content string) (*SpawnPlan, error) {
	content = normalizePlanContent(content)

	var plan SpawnPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		if looksLikeCLIUsage(content) {
			return nil, fmt.Errorf("failed to parse plan JSON: model returned CLI help text instead of JSON; check ~/.orion.yaml llm.base_url and llm.model\nContent: %s", content)
		}
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

func normalizePlanContent(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	if jsonBody := extractFirstJSONObject(content); jsonBody != "" {
		return jsonBody
	}

	return content
}

func extractFirstJSONObject(content string) string {
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}

	inString := false
	escaped := false
	depth := 0
	for i := start; i < len(content); i++ {
		ch := content[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(content[start : i+1])
			}
		}
	}

	return ""
}

func looksLikeCLIUsage(content string) bool {
	s := strings.TrimSpace(strings.ToLower(content))
	return strings.HasPrefix(s, "usage:") ||
		(strings.Contains(s, "usage:") && strings.Contains(s, "flags:"))
}

func logLLMRequest(model, systemPrompt, userPrompt string, temperature float64, maxTokens int) {
	log.Info("ai: LLM request model=%q temperature=%.2f max_tokens=%d system_prompt=%q user_prompt=%q", model, temperature, maxTokens, systemPrompt, userPrompt)
}

func logLLMResponse(model, content string) {
	log.Info("ai: LLM response model=%q content=%q", model, content)
}
