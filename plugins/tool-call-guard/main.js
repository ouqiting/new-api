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
    console.log(
      JSON.stringify({
        action: 'deny',
        code: 403,
        error: 'Tool calls are only available to admin users.',
      })
    );
  } else {
    console.log(JSON.stringify({ action: 'allow' }));
  }
});
