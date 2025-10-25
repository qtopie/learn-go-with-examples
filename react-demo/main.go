/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"golang.org/x/net/proxy"
	"google.golang.org/genai"
)

type QueryWeatherParams struct {
	City *string `json:"city,omitempty" jsonschema:"description=City"`
}

// 处理函数
func QueryWeatherFunc(_ context.Context, params *QueryWeatherParams) (string, error) {
	log.Println("querying weather")
	reqUrl := "https://wttr.in/?T"
	if len(*params.City) > 0 {
		reqUrl = "https://wttr.in/" + url.PathEscape(*params.City) + "?T"
		log.Println("request url", reqUrl)
	}
	resp, err := http.DefaultClient.Get(reqUrl)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	reply := string(data)
	log.Println("weather of", params.City, reply)
	return reply, nil
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
		Model:  "gemini-2.5-flash",
	})
	if err != nil {
		panic(err)
	}

	// prepare persona (system prompt) (optional)
	persona := `# Character:
你是聪明的个人助理，擅长分析数据帮伙伴解决问题
`

	// 使用 InferTool 创建工具
	updateTool, err := utils.InferTool(
		"query_weather", // tool name
		"A tool to query weather, will use default location if no city provide", // tool description
		QueryWeatherFunc)
	if err != nil {
		panic(err)
	}

	ragent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{updateTool},
		},
		// StreamToolCallChecker: toolCallChecker, // uncomment it to replace the default tool call checker with custom one
	})
	if err != nil {
		panic(err)
	}

	opt := []agent.AgentOption{
		agent.WithComposeOptions(compose.WithCallbacks(&LoggerCallback{})),
		//react.WithChatModelOptions(ark.WithCache(cacheOption)),
	}

	sr, err := ragent.Stream(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: persona,
		},
		{
			Role:    schema.User,
			Content: "我在广州，明天适合出门吗？需要注意什么",
		},
	}, opt...)
	if err != nil {
		log.Printf("failed to stream: %v", err)
		return
	}

	defer sr.Close() // remember to close the stream

	log.Printf("\n\n===== start streaming =====\n\n")

	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			// error
			log.Printf("failed to recv: %v", err)
			return
		}

		// 打字机打印
		log.Printf("%v", msg.Content)
	}

	log.Printf("\n\n===== finished =====\n")
	time.Sleep(2 * time.Second)
}

type LoggerCallback struct {
	callbacks.HandlerBuilder // 可以用 callbacks.HandlerBuilder 来辅助实现 callback
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	fmt.Println("==================")
	inputStr, _ := json.MarshalIndent(input, "", "  ")
	fmt.Printf("[OnStart] %s\n", string(inputStr))
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	fmt.Println("=========[OnEnd]=========")
	outputStr, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outputStr))
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Println("=========[OnError]=========")
	fmt.Println(err)
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {

	var graphInfoName = react.GraphName

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[OnEndStream] panic err:", err)
			}
		}()

		defer output.Close() // remember to close the stream in defer

		fmt.Println("=========[OnEndStream]=========")
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			s, err := json.Marshal(frame)
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			if info.Name == graphInfoName { // 仅打印 graph 的输出, 否则每个 stream 节点的输出都会打印一遍
				fmt.Printf("%s: %s\n", info.Name, string(s))
			}
		}

	}()
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	return ctx
}
