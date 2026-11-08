#!/usr/bin/env node

/**
 * ci-upstream-agent.mjs
 *
 * Automated Upstream Convergence & Verification Agent for:
 * CLIProxyAPI (router-for-me/CLIProxyAPI)
 *
 * Features:
 * - Upstream Git fetch and merge with conflict detection
 * - Agentic Conflict Resolution repair loop against OpenAI-compatible LLM endpoints
 * - Verification gate: gofmt -w ., go build ./cmd/server, go test ./...
 * - CI synchronization runner for GitHub Actions
 *
 * Zero-dependency: Node.js >= 20 built-ins only.
 */

import fs from 'node:fs';
import path from 'node:path';
import cp from 'node:child_process';
import process from 'node:process';
import { pathToFileURL } from 'node:url';

/**
 * Standard OpenAI tools exposed to the agent.
 */
export const AGENT_TOOLS = [
  {
    type: 'function',
    function: {
      name: 'read_file',
      description: 'Read contents of a file in the workspace',
      parameters: {
        type: 'object',
        properties: {
          path: { type: 'string', description: 'Relative or absolute file path to read' }
        },
        required: ['path']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'write_file',
      description: 'Write or overwrite a file in the workspace',
      parameters: {
        type: 'object',
        properties: {
          path: { type: 'string', description: 'Relative or absolute file path to write' },
          content: { type: 'string', description: 'Full file content to write' }
        },
        required: ['path', 'content']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'run_command',
      description: 'Execute a shell command in the workspace',
      parameters: {
        type: 'object',
        properties: {
          command: { type: 'string', description: 'Shell command to execute' }
        },
        required: ['command']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'git_status',
      description: 'Inspect git status and uncommitted/conflicted files',
      parameters: {
        type: 'object',
        properties: {}
      }
    }
  }
];

/**
 * Default tool executor for local workspace operations.
 */
export const DEFAULT_TOOL_EXECUTOR = {
  async read_file({ path: filePath }) {
    const fullPath = path.isAbsolute(filePath) ? filePath : path.resolve(process.cwd(), filePath);
    try {
      return fs.readFileSync(fullPath, 'utf8');
    } catch (err) {
      return `Error reading file ${filePath}: ${err.message}`;
    }
  },
  async write_file({ path: filePath, content }) {
    const fullPath = path.isAbsolute(filePath) ? filePath : path.resolve(process.cwd(), filePath);
    try {
      fs.mkdirSync(path.dirname(fullPath), { recursive: true });
      fs.writeFileSync(fullPath, content, 'utf8');
      return `Successfully wrote ${Buffer.byteLength(content, 'utf8')} bytes to ${filePath}`;
    } catch (err) {
      return `Error writing file ${filePath}: ${err.message}`;
    }
  },
  async run_command({ command }) {
    try {
      const stdout = cp.execSync(command, {
        cwd: process.cwd(),
        encoding: 'utf8',
        stdio: ['ignore', 'pipe', 'pipe'],
        maxBuffer: 10 * 1024 * 1024
      });
      return stdout || '(command completed with no output)';
    } catch (err) {
      return `Command failed (exit code ${err.status}):\nSTDOUT:\n${err.stdout || ''}\nSTDERR:\n${err.stderr || ''}`;
    }
  },
  async git_status() {
    try {
      const status = cp.execSync('git status --porcelain', { cwd: process.cwd(), encoding: 'utf8' });
      const diffStat = cp.execSync('git diff --stat', { cwd: process.cwd(), encoding: 'utf8' });
      return `--- git status --porcelain ---\n${status || '(clean)'}\n--- git diff --stat ---\n${diffStat || '(none)'}`;
    } catch (err) {
      return `Git command failed: ${err.message}`;
    }
  }
};

/**
 * Run the agentic repair loop against an OpenAI-compatible endpoint.
 *
 * @param {{
 *   baseUrl: string,
 *   apiKey?: string,
 *   model?: string,
 *   maxTurns?: number,
 *   contextSummary?: string,
 *   verifyFn: () => Promise<{ ok: boolean, output?: string }>,
 *   toolExecutor?: Record<string, Function>,
 *   fetchFn?: typeof globalThis.fetch
 * }} options
 * @returns {Promise<{ resolved: boolean, turns: number, lastError?: string }>}
 */
export async function runAgenticRepairLoop({
  baseUrl,
  apiKey,
  model = 'gemini-3.8-flash-high',
  maxTurns = 10,
  contextSummary = '',
  verifyFn,
  toolExecutor,
  fetchFn = globalThis.fetch
}) {
  let normalizedBase = (baseUrl || 'https://antigravity-manager-963915716511.asia-northeast2.run.app/v1').replace(/\/+$/, '');
  if (!normalizedBase.endsWith('/v1')) {
    normalizedBase += '/v1';
  }
  const completionsUrl = `${normalizedBase}/chat/completions`;

  const systemPrompt = `You are an automated upstream convergence repair agent for CLIProxyAPI.
You must obey the project conventions in AGENTS.md, docs/adr/0002-cloud-run-magpie-unified-gateway.md, and docs/adr/0003-automated-dual-upstream-convergence.md:
- Keep changes small and simple (KISS).
- Write comments in English only.
- Do NOT use github.com/tidwall/gjson, github.com/tidwall/sjson, or Go standard encoding/base64. Always use github.com/MoeForge/moego/ojson and github.com/MoeForge/moego/base64.
- Payload configuration MUST be the final semantic barrier before sending requests in every executor.
- Format all Go files with gofmt -w .
- Resolve any git conflict markers (<<<<<<<, =======, >>>>>>>).
- Fix compilation or test errors until the verification suite passes.`;

  const initialUserPrompt = `Automated upstream convergence verification failed or encountered conflicts.
Context Summary:
${contextSummary}

Please inspect the workspace, resolve all conflicts and compilation/test failures, and fix any broken code.`;

  const messages = [
    { role: 'system', content: systemPrompt },
    { role: 'user', content: initialUserPrompt }
  ];

  let lastError = '';

  for (let turn = 1; turn <= maxTurns; turn++) {
    const headers = {
      'Content-Type': 'application/json'
    };
    if (apiKey) {
      headers['Authorization'] = `Bearer ${apiKey}`;
    }

    const payload = {
      model,
      messages,
      tools: AGENT_TOOLS,
      tool_choice: 'auto'
    };

    const res = await fetchFn(completionsUrl, {
      method: 'POST',
      headers,
      body: JSON.stringify(payload)
    });

    if (!res.ok) {
      const errText = await res.text().catch(() => '');
      throw new Error(`Agent repair API request failed (${res.status} ${res.statusText}): ${errText}`);
    }

    const data = await res.json();
    const choice = data.choices?.[0];
    if (!choice || !choice.message) {
      throw new Error('Invalid response from agent API: no message choice');
    }

    const message = choice.message;
    messages.push(message);

    const toolCalls = message.tool_calls;
    if (Array.isArray(toolCalls) && toolCalls.length > 0) {
      for (const call of toolCalls) {
        const fnName = call.function?.name;
        let fnArgs = {};
        try {
          fnArgs = typeof call.function?.arguments === 'string'
            ? JSON.parse(call.function.arguments)
            : (call.function?.arguments || {});
        } catch {
          fnArgs = { raw: call.function?.arguments };
        }

        const executor = (toolExecutor && toolExecutor[fnName])
          ? toolExecutor[fnName]
          : DEFAULT_TOOL_EXECUTOR[fnName];

        let result = '';
        if (typeof executor === 'function') {
          try {
            result = await executor(fnArgs);
          } catch (execErr) {
            result = `Tool execution error: ${execErr.message}`;
          }
        } else {
          result = `Unknown tool: ${fnName}`;
        }

        messages.push({
          role: 'tool',
          tool_call_id: call.id,
          name: fnName,
          content: typeof result === 'string' ? result : JSON.stringify(result)
        });
      }

      // Verify state after tool execution turn
      const verifyResult = await verifyFn();
      if (verifyResult.ok) {
        return { resolved: true, turns: turn };
      }
      lastError = verifyResult.output || '';
    } else {
      // Agent finished with plain message
      const verifyResult = await verifyFn();
      if (verifyResult.ok) {
        return { resolved: true, turns: turn };
      }
      lastError = verifyResult.output || '';
      messages.push({
        role: 'user',
        content: `Verification gate still failed:\n${lastError}\nPlease fix the remaining errors.`
      });
    }
  }

  return { resolved: false, turns: maxTurns, lastError };
}

/**
 * Standard Go compilation and test verification gate.
 */
export async function runVerification({ cwd = process.cwd() } = {}) {
  try {
    // 1. gofmt -w .
    cp.execSync('gofmt -w .', { cwd, stdio: ['ignore', 'pipe', 'pipe'] });

    // 2. go build ./cmd/server
    const testBinName = `verify-bin-${Date.now()}${process.platform === 'win32' ? '.exe' : ''}`;
    const testBinPath = path.join(cwd, testBinName);
    try {
      cp.execSync(`go build -o ${testBinName} ./cmd/server`, {
        cwd,
        stdio: ['ignore', 'pipe', 'pipe']
      });
    } finally {
      try {
        fs.rmSync(testBinPath, { force: true });
      } catch {
        // ignore removal error
      }
    }

    // 3. go test ./...
    const testOut = cp.execSync('go test ./...', {
      cwd,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe']
    });

    return { ok: true, output: testOut };
  } catch (err) {
    return {
      ok: false,
      output: `${err.message}\nSTDOUT:\n${err.stdout || ''}\nSTDERR:\n${err.stderr || ''}`
    };
  }
}

/**
 * Write key-value pairs to GITHUB_OUTPUT if set.
 */
export function setGithubOutput(entries, env = process.env) {
  const githubOutput = env.GITHUB_OUTPUT;
  if (!githubOutput) return;

  const content = Object.entries(entries)
    .map(([k, v]) => `${k}=${String(v)}`)
    .join('\n') + '\n';

  fs.appendFileSync(githubOutput, content, 'utf8');
}

/**
 * Main CLI entrypoint.
 */
export async function cliMain(args = process.argv.slice(2)) {
  const isSyncCi = args.includes('--sync-ci');

  if (isSyncCi) {
    console.log('[sync-ci] Starting Upstream Sync Cycle for CLIProxyAPI...');
    const repoRoot = process.cwd();
    let changed = false;
    let verified = false;
    let summary = '';

    // Step 1: Git fetch and merge upstream router-for-me/CLIProxyAPI main
    let mergeHadConflicts = false;
    let gitMerged = false;
    try {
      console.log('[sync-ci] Fetching upstream router-for-me/CLIProxyAPI main...');
      cp.execSync('git fetch https://github.com/router-for-me/CLIProxyAPI.git main', { stdio: 'inherit' });

      // Check if FETCH_HEAD is already an ancestor of HEAD
      let isAncestor = false;
      try {
        cp.execSync('git merge-base --is-ancestor FETCH_HEAD HEAD');
        isAncestor = true;
      } catch {
        isAncestor = false;
      }

      if (!isAncestor) {
        console.log('[sync-ci] Merging upstream commits with --no-commit --no-ff...');
        try {
          cp.execSync('git merge --no-commit --no-ff FETCH_HEAD', { stdio: 'inherit' });
          gitMerged = true;
        } catch (mergeErr) {
          console.warn('[sync-ci] Merge produced conflicts.');
          mergeHadConflicts = true;
        }
      } else {
        console.log('[sync-ci] Upstream CLIProxyAPI is already up-to-date.');
      }
    } catch (err) {
      console.warn(`[sync-ci] Upstream fetch/merge warning: ${err.message}`);
    }

    // Step 2: Run initial verification
    console.log('[sync-ci] Running initial verification gate...');
    const initialVerify = await runVerification({ cwd: repoRoot });

    if (!mergeHadConflicts && initialVerify.ok) {
      console.log('[sync-ci] All verification checks passed cleanly without repair.');
      verified = true;
      summary = 'Upstream convergence verified cleanly without conflicts.';
    } else {
      console.log('[sync-ci] Verification failed or merge had conflicts. Invoking Agentic Conflict Resolution...');
      const repairContext = [
        `Merge conflicts encountered: ${mergeHadConflicts}`,
        `Verification output:\n${initialVerify.output || '(none)'}`
      ].filter(Boolean).join('\n\n');

      try {
        const repairResult = await runAgenticRepairLoop({
          baseUrl: process.env.UPSTREAM_AGENT_BASE_URL || 'https://antigravity-manager-963915716511.asia-northeast2.run.app/v1',
          apiKey: process.env.UPSTREAM_AGENT_API_KEY,
          model: process.env.UPSTREAM_AGENT_MODEL || 'gemini-3.8-flash-high',
          maxTurns: 10,
          contextSummary: repairContext,
          verifyFn: () => runVerification({ cwd: repoRoot })
        });

        verified = repairResult.resolved;
        if (verified) {
          summary = `Agentic repair succeeded in ${repairResult.turns} turns.`;
          console.log(`[sync-ci] ${summary}`);
        } else {
          summary = `Agentic repair failed after ${repairResult.turns} turns. Last error: ${repairResult.lastError}`;
          console.error(`[sync-ci] ${summary}`);
        }
      } catch (agentErr) {
        verified = false;
        summary = `Agentic repair encountered an unexpected error: ${agentErr.message}`;
        console.error(`[sync-ci] ${summary}`);
      }
    }

    // Step 3: Detect any uncommitted or changed files
    try {
      const gitStatus = cp.execSync('git status --porcelain', { encoding: 'utf8', cwd: repoRoot }).trim();
      changed = gitStatus.length > 0 || gitMerged;
    } catch {
      changed = gitMerged;
    }

    setGithubOutput({
      changed: changed ? 'true' : 'false',
      verified: verified ? 'true' : 'false',
      summary: summary.replace(/\r?\n/g, ' ')
    });

    console.log(`[sync-ci] Result: changed=${changed}, verified=${verified}`);
    return;
  }

  console.log(`Usage: node scripts/ci-upstream-agent.mjs [options]
Options:
  --sync-ci        Execute Upstream Sync Cycle for CI
`);
}

// Auto-run if executed directly
const isDirectRun = Boolean(
  process.argv[1] &&
  (import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href ||
   import.meta.url === pathToFileURL(process.argv[1]).href)
);

if (isDirectRun) {
  cliMain().catch(err => {
    console.error(`[ci-upstream-agent] Fatal error: ${err.message}`);
    process.exit(1);
  });
}
