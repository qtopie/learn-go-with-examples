# Minimal MCP server (example)

使用Initialize方法初始化服务

```Plaintext
Content-Length: 78

{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"0.1"},"id":1}
```

调用执行工具

```Plaintext
Content-Length: 173

{"jsonrpc":"2.0","method":"tool/use","params":{"toolName":"get_weather","requestID":"req-001","input":{"location":"San Francisco, CA","unit":"fahrenheit"}},"id":2}
```