package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"golang.org/x/net/proxy"
	"google.golang.org/genai"
)

// 参数结构体
type TodoUpdateParams struct {
	ID        string  `json:"id" jsonschema:"description=id of the todo"`
	Content   *string `json:"content,omitempty" jsonschema:"description=content of the todo"`
	StartedAt *int64  `json:"started_at,omitempty" jsonschema:"description=start time in unix timestamp"`
	Deadline  *int64  `json:"deadline,omitempty" jsonschema:"description=deadline of the todo in unix timestamp"`
	Done      *bool   `json:"done,omitempty" jsonschema:"description=done status"`
}

type TodoAddParams struct {
	ID        string  `json:"id" jsonschema:"description=id of the todo"`
	Content   *string `json:"content,omitempty" jsonschema:"description=content of the todo"`
	StartedAt *int64  `json:"started_at,omitempty" jsonschema:"description=start time in unix timestamp"`
	Deadline  *int64  `json:"deadline,omitempty" jsonschema:"description=deadline of the todo in unix timestamp"`
	Done      *bool   `json:"done,omitempty" jsonschema:"description=done status"`
}

// 处理函数
func UpdateTodoFunc(_ context.Context, params *TodoUpdateParams) (string, error) {
	// Mock处理逻辑
	return `{"msg": "update todo success"}`, nil
}

// 处理函数
func AddTodoFunc(_ context.Context, params *TodoAddParams) (string, error) {
	// Mock处理逻辑
	return `{"msg": "add todo success"}`, nil
}

func getAddTodoTool() tool.InvokableTool {
	// 工具信息
	info := &schema.ToolInfo{
		Name: "add_todo",
		Desc: "Add a todo item",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"content": {
				Desc:     "The content of the todo item",
				Type:     schema.String,
				Required: true,
			},
			"started_at": {
				Desc: "The started time of the todo item, in unix timestamp",
				Type: schema.Integer,
			},
			"deadline": {
				Desc: "The deadline of the todo item, in unix timestamp",
				Type: schema.Integer,
			},
		}),
	}

	// 使用NewTool创建工具
	return utils.NewTool(info, AddTodoFunc)
}

func main() {
	ctx := context.Background()

	// SOCKS proxy address
	proxyURL, err := url.Parse("socks5://127.0.0.1:1080")
	if err != nil {
		panic(err)
	}

	// Create a SOCKS5 dialer
	dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		panic(err)
	}

	// Create HTTP client with the SOCKS5 dialer
	httpTransport := &http.Transport{
		Dial: dialer.Dial,
	}
	httpClient := &http.Client{Transport: httpTransport}


	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:     os.Getenv("GEMINI_API_TOKEN"),
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: httpClient,
	})
	if err != nil {
		log.Fatal(err)
	}

	chatModel, err := gemini.NewChatModel(context.Background(), &gemini.Config{
		Client: client,
		Model:  "gemini-2.0-flash",
	})
	if err != nil {
		panic(err)
	}

	// 使用 InferTool 创建工具
	updateTool, err := utils.InferTool(
		"update_todo", // tool name
		"Update a todo item, eg: content,deadline...", // tool description
		UpdateTodoFunc)
	if err != nil {
		panic(err)
	}

	// // 创建 duckduckgo Search 工具
	searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{
		ToolName:   "duckduckgo_search",                        // 工具名称
		ToolDesc:   "search web for information by duckduckgo", // 工具描述
		Region:     duckduckgo.RegionWT,                        // 搜索地区
		MaxResults: 10,                                         // 每页结果数量
		HTTPClient: httpClient,
	})
	if err != nil {
		panic(err)
	}

	// 初始化 tools
	todoTools := []tool.BaseTool{
		getAddTodoTool(), // NewTool 构建
		updateTool,       // InferTool 构建
		searchTool,       // 官方封装的工具
	}

	// 获取工具信息并绑定到 ChatModel
	toolInfos := make([]*schema.ToolInfo, 0, len(todoTools))
	for _, tool := range todoTools {
		info, err := tool.Info(ctx)
		if err != nil {
			log.Fatal(err)
		}
		toolInfos = append(toolInfos, info)
	}
	err = chatModel.BindTools(toolInfos)
	if err != nil {
		log.Fatal(err)
	}

	// 创建 tools 节点
	todoToolsNode, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
		Tools: todoTools,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Init eino devops server
	err = devops.Init(ctx)
	if err != nil {
		panic(err)
	}

	// 构建完整的处理链
	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.
		AppendChatModel(chatModel, compose.WithNodeName("chat_model")).
		AppendToolsNode(todoToolsNode, compose.WithNodeName("tools"))

	// 编译并运行 chain
	agent, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 运行示例
	resp, err := agent.Invoke(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "添加一个学习 Eino 的 TODO",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 输出结果
	for _, msg := range resp {
		fmt.Println(msg.Content)
	}
}
