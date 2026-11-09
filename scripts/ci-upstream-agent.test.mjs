import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';

import {
  AGENT_TOOLS,
  DEFAULT_TOOL_EXECUTOR,
  fetchLatestUpstreamRelease,
  runAgenticRepairLoop,
  setGithubOutput,
  cliMain
} from './ci-upstream-agent.mjs';

test('fetchLatestUpstreamRelease: extracts latest release tag from GitHub API', async () => {
  const mockFetch = async (url) => {
    assert.equal(url, 'https://api.github.com/repos/router-for-me/CLIProxyAPI/releases/latest');
    return {
      ok: true,
      status: 200,
      json: async () => ({
        tag_name: 'v8.0.61',
        name: 'v8.0.61',
        published_at: '2026-11-08T14:31:53Z',
        html_url: 'https://github.com/router-for-me/CLIProxyAPI/releases/tag/v8.0.61'
      })
    };
  };

  const rel = await fetchLatestUpstreamRelease({ fetchFn: mockFetch });
  assert.equal(rel.tagName, 'v8.0.61');
  assert.equal(rel.publishedAt, '2026-11-08T14:31:53Z');
});

test('fetchLatestUpstreamRelease: throws on HTTP error or missing tag_name', async () => {
  await assert.rejects(
    async () => fetchLatestUpstreamRelease({
      fetchFn: async () => ({ ok: false, status: 404, statusText: 'Not Found' })
    }),
    /Failed to fetch latest release/
  );
  await assert.rejects(
    async () => fetchLatestUpstreamRelease({
      fetchFn: async () => ({ ok: true, status: 200, json: async () => ({}) })
    }),
    /did not contain a valid tag_name/
  );
});

test('AGENT_TOOLS: contains expected tool definitions and schemas', () => {
  const toolNames = AGENT_TOOLS.map(t => t.function.name);
  assert.deepEqual(toolNames.sort(), ['git_status', 'read_file', 'run_command', 'write_file']);

  for (const tool of AGENT_TOOLS) {
    assert.equal(tool.type, 'function');
    assert.ok(tool.function.description);
    assert.equal(tool.function.parameters.type, 'object');
  }
});

test('DEFAULT_TOOL_EXECUTOR: read_file and write_file operations', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'tool-test-'));
  try {
    const testFile = path.join(tempDir, 'sub', 'test.txt');
    const content = 'hello world from tool executor';

    const writeRes = await DEFAULT_TOOL_EXECUTOR.write_file({ path: testFile, content });
    assert.match(writeRes, /Successfully wrote \d+ bytes/);
    assert.equal(fs.readFileSync(testFile, 'utf8'), content);

    const readRes = await DEFAULT_TOOL_EXECUTOR.read_file({ path: testFile });
    assert.equal(readRes, content);

    const nonExistentRes = await DEFAULT_TOOL_EXECUTOR.read_file({ path: path.join(tempDir, 'missing.txt') });
    assert.match(nonExistentRes, /Error reading file/);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test('DEFAULT_TOOL_EXECUTOR: run_command executes basic shell commands', async () => {
  const isWindows = process.platform === 'win32';
  const cmd = isWindows ? 'cmd /c echo tool_command_success' : 'echo tool_command_success';
  const out = await DEFAULT_TOOL_EXECUTOR.run_command({ command: cmd });
  assert.match(out, /tool_command_success/);
});

test('setGithubOutput: appends output variables or handles unset GITHUB_OUTPUT', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gh-output-test-'));
  try {
    const outputFile = path.join(tempDir, 'output.txt');
    fs.writeFileSync(outputFile, '', 'utf8');

    setGithubOutput({ changed: 'true', verified: 'true', summary: 'all good' }, { GITHUB_OUTPUT: outputFile });

    const content = fs.readFileSync(outputFile, 'utf8');
    assert.match(content, /changed=true/);
    assert.match(content, /verified=true/);
    assert.match(content, /summary=all good/);

    // No-op when GITHUB_OUTPUT is empty
    assert.doesNotThrow(() => setGithubOutput({ changed: 'false' }, {}));
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test('cliMain: prints usage when invoked with empty or unrecognized args', async () => {
  const logs = [];
  const originalLog = console.log;
  console.log = (...args) => logs.push(args.join(' '));
  try {
    await cliMain([]);
    assert.ok(logs.some(l => l.includes('Usage: node scripts/ci-upstream-agent.mjs')));
  } finally {
    console.log = originalLog;
  }
});

test('runAgenticRepairLoop: 1-turn repair stops immediately on green verification', async () => {
  const calls = [];
  const mockFetch = async (url, opts) => {
    calls.push({ url, opts });
    return {
      ok: true,
      status: 200,
      json: async () => ({
        choices: [
          {
            message: {
              role: 'assistant',
              content: 'Fixing the build error by writing code',
              tool_calls: [
                {
                  id: 'call_1',
                  type: 'function',
                  function: {
                    name: 'write_file',
                    arguments: JSON.stringify({ path: 'fixed.txt', content: 'resolved' })
                  }
                }
              ]
            }
          }
        ]
      })
    };
  };

  let fileWritten = false;
  const mockToolExecutor = {
    write_file: async () => {
      fileWritten = true;
      return 'ok';
    }
  };

  let verifyCallCount = 0;
  const verifyFn = async () => {
    verifyCallCount++;
    return { ok: fileWritten };
  };

  const result = await runAgenticRepairLoop({
    baseUrl: 'https://test-agent.example.com',
    apiKey: 'secret-key',
    model: 'gemini-3.8-flash-high',
    maxTurns: 5,
    contextSummary: 'Build broken on line 12',
    verifyFn,
    toolExecutor: mockToolExecutor,
    fetchFn: mockFetch
  });

  assert.equal(result.resolved, true);
  assert.equal(result.turns, 1);
  assert.equal(fileWritten, true);
  assert.equal(verifyCallCount, 1);
  assert.equal(calls[0].url, 'https://test-agent.example.com/v1/chat/completions');
  assert.equal(calls[0].opts.headers['Authorization'], 'Bearer secret-key');
});

test('runAgenticRepairLoop: completion message with no tool calls and passing verifyFn', async () => {
  const mockFetch = async () => ({
    ok: true,
    status: 200,
    json: async () => ({
      choices: [
        {
          message: {
            role: 'assistant',
            content: 'The workspace is already in order.'
          }
        }
      ]
    })
  });

  const verifyFn = async () => ({ ok: true, output: 'passed' });

  const result = await runAgenticRepairLoop({
    baseUrl: 'https://test-agent.example.com/v1',
    maxTurns: 5,
    verifyFn,
    fetchFn: mockFetch
  });

  assert.equal(result.resolved, true);
  assert.equal(result.turns, 1);
});

test('runAgenticRepairLoop: multi-turn repair reaches resolution on turn 2', async () => {
  let turnCounter = 0;
  const mockFetch = async () => {
    turnCounter++;
    if (turnCounter === 1) {
      return {
        ok: true,
        status: 200,
        json: async () => ({
          choices: [
            {
              message: {
                role: 'assistant',
                tool_calls: [
                  {
                    id: 'call_turn1',
                    type: 'function',
                    function: {
                      name: 'read_file',
                      arguments: JSON.stringify({ path: 'main.go' })
                    }
                  }
                ]
              }
            }
          ]
        })
      };
    }
    return {
      ok: true,
      status: 200,
      json: async () => ({
        choices: [
          {
            message: {
              role: 'assistant',
              tool_calls: [
                {
                  id: 'call_turn2',
                  type: 'function',
                  function: {
                    name: 'write_file',
                    arguments: JSON.stringify({ path: 'main.go', content: 'package main' })
                  }
                }
              ]
            }
          }
        ]
      })
    };
  };

  let isFixed = false;
  const mockToolExecutor = {
    read_file: async () => 'broken content',
    write_file: async () => {
      isFixed = true;
      return 'written';
    }
  };

  const verifyFn = async () => {
    return isFixed ? { ok: true } : { ok: false, output: 'syntax error' };
  };

  const result = await runAgenticRepairLoop({
    baseUrl: 'https://test-agent.example.com/v1',
    apiKey: 'test-key',
    maxTurns: 5,
    verifyFn,
    toolExecutor: mockToolExecutor,
    fetchFn: mockFetch
  });

  assert.equal(result.resolved, true);
  assert.equal(result.turns, 2);
});

test('runAgenticRepairLoop: exhausts maxTurns and reports failure', async () => {
  const mockFetch = async () => ({
    ok: true,
    status: 200,
    json: async () => ({
      choices: [
        {
          message: {
            role: 'assistant',
            content: 'Still trying to fix...',
            tool_calls: [
              {
                id: 'call_retry',
                type: 'function',
                function: {
                  name: 'run_command',
                  arguments: JSON.stringify({ command: 'echo retry' })
                }
              }
            ]
          }
        }
      ]
    })
  });

  const mockToolExecutor = {
    run_command: async () => 'retry'
  };

  const verifyFn = async () => ({ ok: false, output: 'fatal compile error' });

  const result = await runAgenticRepairLoop({
    baseUrl: 'https://test-agent.example.com/v1',
    apiKey: 'test-key',
    maxTurns: 3,
    verifyFn,
    toolExecutor: mockToolExecutor,
    fetchFn: mockFetch
  });

  assert.equal(result.resolved, false);
  assert.equal(result.turns, 3);
  assert.equal(result.lastError, 'fatal compile error');
});

test('runAgenticRepairLoop: throws when API response is HTTP error', async () => {
  const mockFetch = async () => ({
    ok: false,
    status: 500,
    statusText: 'Internal Server Error',
    text: async () => 'upstream crashed'
  });

  await assert.rejects(
    async () => runAgenticRepairLoop({
      baseUrl: 'https://test-agent.example.com/v1',
      maxTurns: 2,
      verifyFn: async () => ({ ok: false }),
      fetchFn: mockFetch
    }),
    /Agent repair API request failed \(500 Internal Server Error\): upstream crashed/
  );
});
