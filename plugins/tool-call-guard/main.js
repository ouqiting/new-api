const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false,
});

const DENY_MESSAGE = '普通用户禁止调用工具，如有误判请联系管理员。';

const TOOL_FIELD_KEYS = new Set([
  'tools',
  'tool_choice',
  'toolChoice',
  'tool_config',
  'toolConfig',
  'functions',
  'function_call',
  'functionCall',
  'function_response',
  'functionResponse',
  'function_declarations',
  'functionDeclarations',
  'tool_calls',
  'tool_call_id',
  'tool_use_id',
  'mcp_servers',
  'mcpServers',
  'parallel_tool_calls',
  'max_tool_calls',
]);

const TOOL_TYPE_VALUES = new Set([
  'tool',
  'function',
  'tool_use',
  'tool_result',
  'function_call',
  'function_call_output',
  'function_response',
  'mcp_call',
  'mcp_list_tools',
  'web_search_call',
  'computer_call',
  'code_interpreter_call',
  'local_shell_call',
]);

function hasMeaningfulValue(value) {
  if (value === null || value === undefined) {
    return false;
  }
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  if (typeof value === 'object') {
    return Object.keys(value).length > 0;
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase();
    return normalized !== '' && normalized !== 'none' && normalized !== 'null';
  }
  return true;
}

function containsToolCallFormat(value) {
  if (Array.isArray(value)) {
    return value.some((item) => containsToolCallFormat(item));
  }

  if (value === null || typeof value !== 'object') {
    return false;
  }

  for (const [key, child] of Object.entries(value)) {
    if (TOOL_FIELD_KEYS.has(key) && hasMeaningfulValue(child)) {
      return true;
    }

    if ((key === 'role' || key === 'type') && typeof child === 'string') {
      if (TOOL_TYPE_VALUES.has(child.trim().toLowerCase())) {
        return true;
      }
    }

    if (containsToolCallFormat(child)) {
      return true;
    }
  }

  return false;
}

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

  if (!isAdmin && containsToolCallFormat(requestBody)) {
    console.log(
      JSON.stringify({
        action: 'deny',
        code: 400,
        error: DENY_MESSAGE,
        log: true,
        logContent: '插件「Tool Call Guard」拦截了普通用户的工具调用请求',
        logDetail: {
          reason: 'tool_call_detected',
          role: role || 'user',
          model: event.context?.model || '',
        },
      })
    );
  } else {
    console.log(JSON.stringify({ action: 'allow' }));
  }
});
