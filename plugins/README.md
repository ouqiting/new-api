# 插件系统

本文档说明 new-api 插件系统的设计规范、开发方式和使用方式。

## 概述

插件系统允许管理员通过扩展脚本控制请求生命周期，例如：

- 在请求转发到上游渠道前进行拦截、修改或拒绝；
- 在上游响应返回后处理或记录日志；
- 在服务端启动时执行初始化任务。

插件以后台子进程方式运行，通过标准输入输出（stdin/stdout）以 JSON 行格式与主程序通信。插件拥有较高的权限（可读取请求体、用户上下文并决定是否拒绝请求），因此**仅 root 账号**可以在后台管理插件。

## 插件目录结构

插件统一存放在项目根目录的 `plugins/` 文件夹下。每个插件是一个独立的子目录，至少包含一个清单文件 `plugin.json`：

```
plugins/
├── tool-call-guard/
│   ├── plugin.json
│   └── main.js
└── my-plugin/
│   ├── plugin.json
│   └── main.py
```

插件目录名建议与 `plugin.json` 中的 `id` 保持一致，便于识别。

## plugin.json 字段说明

```json
{
  "id": "tool-call-guard",
  "title": "Tool Call Guard",
  "description": "非管理员用户携带 tool 声明时直接拒绝请求",
  "version": "1.0.0",
  "author": "作者名",
  "entry": "main.js",
  "hooks": ["pre-request"],
  "capabilities": ["read-request", "deny-request"],
  "config": {}
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `id` | 是 | 插件唯一标识，建议与目录名一致。 |
| `title` | 是 | 插件标题，显示在管理后台卡片左上角。 |
| `description` | 是 | 插件描述，显示在卡片标题下方。 |
| `version` | 是 | 插件版本号，建议使用语义化版本。 |
| `author` | 否 | 作者名称。 |
| `entry` | 否 | 入口文件相对路径。如果为空，则插件只作为可启用/停用的标记，不会执行。 |
| `hooks` | 否 | 插件订阅的事件数组，可选值：`startup`、`pre-request`、`post-response`。 |
| `capabilities` | 否 | 插件声明的能力，用于文档和后续权限控制。 |
| `log` | 否 | 是否在插件触发时写入「插件」类别的使用日志（默认 `false`）。可被插件返回结果中的 `log` 字段按次覆盖。 |
| `config` | 否 | 插件默认配置对象，可在后续版本中通过后台调整。 |

## 执行模型

### 进程模型

每个启用的插件会在后台启动一个独立的子进程。入口文件按扩展名自动选择解释器：

| 扩展名 | 运行方式 |
|---|---|
| `.js` | `node <entry>` |
| `.py` | `python <entry>` |
| `.sh` | `bash <entry>` |
| 其他 | 直接执行 |

子进程启动后长期保持运行。每次触发事件时，主程序向子进程写入一行 JSON，子进程读取后输出一行 JSON 作为响应。

### 超时与容错

- 单次插件调用超时时间为 **5 秒**；
- 超时后子进程会被强制终止，插件会进入未加载状态；
- 插件崩溃不会影响主程序，但会记录到系统日志；
- 插件返回未知动作时按 `allow` 处理。

## 事件与上下文

### 事件格式

主程序向插件发送的事件：

```json
{
  "hook": "pre-request",
  "context": {
    "userId": 123,
    "role": "admin",
    "group": "default",
    "tokenName": "my-token",
    "model": "gpt-4"
  },
  "request": {
    "model": "gpt-4",
    "messages": [],
    "tools": []
  }
}
```

字段说明：

- `hook`：当前触发的事件类型；
- `context.userId`：用户 ID；
- `context.role`：用户角色，可能的值为 `root`、`admin`、`user`；
- `context.group`：用户所在分组；
- `context.tokenName`：当前使用的 API Key 名称；
- `context.model`：请求模型名称；
- `request`：请求体 JSON，仅在 `pre-request` 和 `post-response` 事件中出现。`post-response` 中对应上游响应内容。

### 支持的事件

| 事件 | 触发时机 | 说明 |
|---|---|---|
| `startup` | 服务端启动并加载插件后 | 用于初始化任务，如注册第三方服务、预热缓存等。 |
| `pre-request` | 请求解析完成后、转发到上游前 | 可读取请求体并决定是否拒绝或修改。 |
| `post-response` | 上游响应返回后 | 可读取响应内容并决定是否修改或记录。 |

## 插件响应格式

插件必须向标准输出写入一行 JSON，包含以下字段：

### 放行

```json
{ "action": "allow" }
```

表示继续正常处理。

### 拒绝

```json
{
  "action": "deny",
  "code": 403,
  "error": "Tool calls are only available to admin users."
}
```

主程序会立即终止请求，并将 `code` 和 `error` 返回给调用方。如果 `code` 未提供，默认使用 `403`。

### 修改

```json
{
  "action": "modify",
  "request": {
    "model": "gpt-3.5-turbo"
  }
}
```

表示将当前请求体/响应体替换为 `request` 字段中的完整 JSON。注意：如果多个插件都返回 `modify`，后执行的插件会覆盖前者。

### 日志字段（可选）

任意动作（`allow` / `deny` / `modify`）都可以附带日志字段，用于在「插件」类别的使用日志中记录本次触发，便于管理员和 root 用户在请求日志中查看是哪位用户触发了哪个插件：

```json
{
  "action": "deny",
  "code": 400,
  "error": "普通用户禁止调用工具",
  "log": true,
  "logContent": "插件「Tool Call Guard」拦截了普通用户的工具调用请求",
  "logDetail": {
    "reason": "tool_call_detected",
    "role": "user"
  }
}
```

| 字段 | 说明 |
|---|---|
| `log` | 是否为本次触发写入日志。省略时使用 `plugin.json` 中的 `log` 值；显式设置 `true`/`false` 可按次覆盖。 |
| `logContent` | 日志内容（`content`）。省略时由插件标题与动作自动生成默认文案。 |
| `logDetail` | 附加详情对象，存入日志的 `other` 字段，仅管理员 / root 在日志详情中可见。 |

日志会记录用户、模型、令牌、分组与请求 ID 等信息，类别为「插件」（`LogTypePlugin`）。

## 开发示例

### 示例 1：拒绝非管理员的 tool call（已内置）

`plugins/tool-call-guard/plugin.json`：

```json
{
  "id": "tool-call-guard",
  "title": "Tool Call Guard",
  "description": "非管理员用户携带 tool 声明时直接拒绝请求",
  "version": "1.0.0",
  "author": "new-api",
  "entry": "main.js",
  "hooks": ["pre-request"],
  "capabilities": ["read-request", "deny-request"]
}
```

`plugins/tool-call-guard/main.js`：

```javascript
const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false,
});

rl.on('line', (line) => {
  let event;
  try {
    event = JSON.parse(line);
  } catch (err) {
    console.log(JSON.stringify({ action: 'allow' }));
    return;
  }

  if (event.hook !== 'pre-request') {
    console.log(JSON.stringify({ action: 'allow' }));
    return;
  }

  const role = event.context?.role || '';
  const isAdmin = role === 'root' || role === 'admin';
  const requestBody = event.request || {};
  const hasTools = Array.isArray(requestBody.tools) && requestBody.tools.length > 0;
  const hasToolChoice = requestBody.tool_choice && requestBody.tool_choice !== 'none';

  if (!isAdmin && (hasTools || hasToolChoice)) {
    console.log(JSON.stringify({
      action: 'deny',
      code: 403,
      error: 'Tool calls are only available to admin users.',
    }));
  } else {
    console.log(JSON.stringify({ action: 'allow' }));
  }
});
```

### 示例 2：记录请求日志

```json
{
  "id": "request-logger",
  "title": "Request Logger",
  "description": "打印每次请求使用的模型",
  "version": "1.0.0",
  "entry": "logger.js",
  "hooks": ["pre-request"],
  "capabilities": ["read-request"]
}
```

```javascript
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin, output: process.stdout, terminal: false });

rl.on('line', (line) => {
  const event = JSON.parse(line);
  console.error(`[${event.context.role}] ${event.context.userId} -> ${event.request?.model || 'unknown'}`);
  console.log(JSON.stringify({ action: 'allow' }));
});
```

> 提示：插件的标准错误（stderr）会写入服务端日志，可用于调试。

## 使用方式

### 安装插件

1. 将插件文件夹复制到 `plugins/` 目录下；
2. 确保 `plugin.json` 有效且入口文件可执行；
3. 以 `root` 账号登录后台；
4. 进入侧边栏 **插件** 页面；
5. 点击右上角 **重新加载**，扫描新插件；
6. 在卡片右下角打开开关启用插件。

### 停用插件

在插件卡片右下角关闭开关即可。后台会停止对应子进程，插件不再生效。

### 调试插件

- 查看服务端日志中的 `plugin <id> stderr:` 和 `plugin <id> exited:` 信息；
- 确保入口文件第一行可正常启动，且不会立即退出；
- 确保插件输出严格为单行 JSON，并以换行符结尾；
- 确保使用 `node` 等解释器时，系统 PATH 中已安装对应运行时。

## 安全与限制

- **权限控制**：仅 `root` 账号可在后台管理插件；
- **进程隔离**：插件运行在独立子进程中，崩溃不会影响主服务；
- **超时保护**：单次调用超过 5 秒会被强制终止；
- **能力声明**：`capabilities` 字段仅用于文档说明，未来可能用于更细粒度的权限控制；
- **数据安全**：插件可以读取完整请求体和响应体，请勿安装来源不明的插件；
- **环境依赖**：`.js` 插件需要服务器已安装 Node.js，`.py` 插件需要 Python。

## 自定义插件目录

默认插件目录为项目根目录下的 `plugins/`。可通过环境变量覆盖：

```bash
PLUGIN_DIR=/path/to/plugins
```
