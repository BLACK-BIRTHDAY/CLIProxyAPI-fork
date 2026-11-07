#!/usr/bin/env node

/**
 * ci-upstream-agent.mjs
 *
 * Automated Dual-Upstream Convergence & Verification Agent for:
 * 1. CLIProxyAPI (router-for-me/CLIProxyAPI)
 * 2. MAGPIE Gemini Web / AIStudio2API (BUKOWSKIREAL/magpie-gemini-web)
 *
 * Features:
 * - Deterministic Studio Release Pin parsing and Dockerfile / process.go updates
 * - Local AIStudio2API binary download, SHA-256 verification, and atomic installation
 * - Agentic Conflict Resolution repair loop against OpenAI-compatible LLM endpoints
 * - Dual-Upstream CI synchronization runner for GitHub Actions
 *
 * Zero-dependency: Node.js >= 20 built-ins only.
 */

import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import cp from 'node:child_process';
import process from 'node:process';
import { pathToFileURL } from 'node:url';

/**
 * Compute SHA-256 hex digest for a Buffer or Uint8Array.
 */
export function sha256Buffer(buffer) {
  return crypto.createHash('sha256').update(buffer).digest('hex').toLowerCase();
}

/**
 * Compute SHA-256 hex digest for a file.
 */
export function sha256File(filePath) {
  const hash = crypto.createHash('sha256');
  const buffer = fs.readFileSync(filePath);
  hash.update(buffer);
  return hash.digest('hex').toLowerCase();
}

/**
 * Extract an archive (.tar.gz, .tgz, .zip) to destDir using available platform utilities.
 */
export function extractArchive(archivePath, destDir) {
  fs.mkdirSync(destDir, { recursive: true });
  const isWindows = process.platform === 'win32';
  const ext = path.extname(archivePath).toLowerCase();

  if (isWindows) {
    const sysTar = 'C:/Windows/System32/tar.exe';
    if (fs.existsSync(sysTar)) {
      cp.execFileSync(sysTar, ['-xf', archivePath, '-C', destDir]);
      return;
    }
    if (ext === '.zip') {
      const escapedArchive = archivePath.replace(/'/g, "''");
      const escapedDest = destDir.replace(/'/g, "''");
      cp.execFileSync('powershell', [
        '-NoProfile',
        '-NonInteractive',
        '-Command',
        `Expand-Archive -LiteralPath '${escapedArchive}' -DestinationPath '${escapedDest}' -Force`
      ]);
      return;
    }
  }

  // Unix or fallback
  if (ext === '.zip') {
    try {
      cp.execFileSync('unzip', ['-q', '-o', archivePath, '-d', destDir]);
    } catch {
      cp.execFileSync('tar', ['-xf', archivePath, '-C', destDir]);
    }
  } else {
    // .tar.gz, .tgz
    cp.execFileSync('tar', ['-xzf', archivePath, '-C', destDir]);
  }
}

/**
 * Recursively search a directory for a file matching binaryName.
 */
export function findBinary(dir, binaryName) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isFile() && entry.name === binaryName) {
      return fullPath;
    }
    if (entry.isDirectory()) {
      const found = findBinary(fullPath, binaryName);
      if (found) return found;
    }
  }
  return null;
}

/**
 * Parse and validate build.json metadata from a BUKOWSKIREAL/magpie-gemini-web release.
 *
 * @param {object|string} buildJson
 * @param {string} [tagName]
 * @returns {{ version: string, tag: string, binaries: Record<string, { asset: string, sha256: string, binarySha256: string }> }}
 */
export function parseStudioBuildJson(buildJson, tagName) {
  const obj = typeof buildJson === 'string' ? JSON.parse(buildJson) : buildJson;
  if (!obj || typeof obj !== 'object') {
    throw new Error('Invalid build.json: must be an object');
  }
  if (!obj.version || typeof obj.version !== 'string') {
    throw new Error('Invalid build.json: missing or invalid "version"');
  }

  const version = obj.version;
  const tag = tagName || (obj.tag || `studio-v${version}`);

  const requiredTargets = ['linux-x64', 'linux-arm64', 'win32-x64', 'darwin-arm64', 'darwin-x64'];
  if (!obj.binaries || typeof obj.binaries !== 'object') {
    throw new Error('Invalid build.json: missing "binaries" object');
  }

  const sha256Regex = /^[a-f0-9]{64}$/i;
  const binaries = {};

  for (const target of requiredTargets) {
    const entry = obj.binaries[target];
    if (!entry || typeof entry !== 'object') {
      throw new Error(`Invalid build.json: missing target "${target}" under binaries`);
    }
    if (!entry.asset || typeof entry.asset !== 'string') {
      throw new Error(`Invalid build.json: target "${target}" missing "asset"`);
    }
    if (!entry.sha256 || !sha256Regex.test(entry.sha256)) {
      throw new Error(`Invalid build.json: target "${target}" missing or invalid 64-hex "sha256"`);
    }
    if (!entry.binarySha256 || !sha256Regex.test(entry.binarySha256)) {
      throw new Error(`Invalid build.json: target "${target}" missing or invalid 64-hex "binarySha256"`);
    }

    binaries[target] = {
      asset: entry.asset,
      sha256: entry.sha256.toLowerCase(),
      binarySha256: entry.binarySha256.toLowerCase()
    };
  }

  return {
    version,
    tag,
    binaries
  };
}

/**
 * Update the Studio Release Pin in Dockerfile and internal/magpie/process.go under repoRoot.
 *
 * @param {{ repoRoot: string, pinInfo: ReturnType<typeof parseStudioBuildJson> }} options
 * @returns {{ changed: boolean, previousVersion: string, newVersion: string }}
 */
export function updateStudioReleasePinFiles({ repoRoot, pinInfo }) {
  const dockerfilePath = path.join(repoRoot, 'Dockerfile');
  const processGoPath = path.join(repoRoot, 'internal', 'magpie', 'process.go');

  if (!fs.existsSync(dockerfilePath)) {
    throw new Error(`Dockerfile not found at ${dockerfilePath}`);
  }
  if (!fs.existsSync(processGoPath)) {
    throw new Error(`process.go not found at ${processGoPath}`);
  }

  const dockerfileContent = fs.readFileSync(dockerfilePath, 'utf8');
  const processGoContent = fs.readFileSync(processGoPath, 'utf8');

  // 1. Check & update internal/magpie/process.go
  const versionMatch = processGoContent.match(/MagpieStudioVersion\s*=\s*"([^"]+)"/);
  if (!versionMatch) {
    throw new Error('Could not find MagpieStudioVersion in internal/magpie/process.go');
  }
  const previousVersion = versionMatch[1];
  const newProcessGoContent = processGoContent.replace(
    /(MagpieStudioVersion\s*=\s*")[^"]+(")/,
    `$1${pinInfo.version}$2`
  );

  // 2. Check & update Dockerfile
  const linuxX64 = pinInfo.binaries['linux-x64'];
  const linuxArm64 = pinInfo.binaries['linux-arm64'];
  const amd64Url = `https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/${pinInfo.tag}/${linuxX64.asset}`;
  const arm64Url = `https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/${pinInfo.tag}/${linuxArm64.asset}`;

  let newDockerfileContent = dockerfileContent.replace(
    /(amd64\|x86_64\)\s*\\\s*[\r\n]+)(\s*URL=")([^"]+)(";\s*\\\s*[\r\n]+)(\s*ARCHIVE_SHA256=")([a-f0-9]+)(";\s*\\\s*[\r\n]+)(\s*BIN_SHA256=")([a-f0-9]+)/g,
    `$1$2${amd64Url}$4$5${linuxX64.sha256}$7$8${linuxX64.binarySha256}`
  );

  newDockerfileContent = newDockerfileContent.replace(
    /(arm64\|aarch64\)\s*\\\s*[\r\n]+)(\s*URL=")([^"]+)(";\s*\\\s*[\r\n]+)(\s*ARCHIVE_SHA256=")([a-f0-9]+)(";\s*\\\s*[\r\n]+)(\s*BIN_SHA256=")([a-f0-9]+)/g,
    `$1$2${arm64Url}$4$5${linuxArm64.sha256}$7$8${linuxArm64.binarySha256}`
  );

  const changed = (newProcessGoContent !== processGoContent) || (newDockerfileContent !== dockerfileContent);

  if (changed) {
    fs.writeFileSync(dockerfilePath, newDockerfileContent, 'utf8');
    fs.writeFileSync(processGoPath, newProcessGoContent, 'utf8');
  }

  return {
    changed,
    previousVersion,
    newVersion: pinInfo.version
  };
}

/**
 * Download, verify SHA-256, and install local AIStudio2API binary into dataDir.
 *
 * @param {{
 *   dataDir: string,
 *   platform?: string,
 *   arch?: string,
 *   pinInfo: ReturnType<typeof parseStudioBuildJson>,
 *   fetchFn?: typeof globalThis.fetch
 * }} options
 * @returns {Promise<{ updated: boolean, version: string, binaryPath: string }>}
 */
export async function installLocalStudioBinary({
  dataDir,
  platform = process.platform,
  arch = process.arch,
  pinInfo,
  fetchFn = globalThis.fetch
}) {
  const platformMap = {
    'win32-x64': 'win32-x64',
    'linux-x64': 'linux-x64',
    'linux-arm64': 'linux-arm64',
    'darwin-arm64': 'darwin-arm64',
    'darwin-x64': 'darwin-x64'
  };

  const key = `${platform}-${arch}`;
  const targetKey = platformMap[key];
  if (!targetKey) {
    throw new Error(`Unsupported platform/architecture: ${platform}/${arch}`);
  }

  const target = pinInfo.binaries?.[targetKey];
  if (!target) {
    throw new Error(`pinInfo missing binaries for target ${targetKey}`);
  }

  const binaryName = platform === 'win32' ? 'aistudio2api.exe' : 'aistudio2api';
  const packageDir = path.join(dataDir, 'package');
  const targetBinaryPath = path.join(packageDir, binaryName);
  const manifestPath = path.join(dataDir, 'install-manifest.json');

  // Fast check: already matching manifest and binary exists
  if (fs.existsSync(manifestPath) && fs.existsSync(targetBinaryPath)) {
    try {
      const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
      if (
        manifest.version === pinInfo.version &&
        manifest.binarySha256?.toLowerCase() === target.binarySha256.toLowerCase()
      ) {
        return {
          updated: false,
          version: pinInfo.version,
          binaryPath: targetBinaryPath
        };
      }
    } catch {
      // Manifest corrupted, re-install
    }
  }

  const downloadUrl = `https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/${pinInfo.tag}/${target.asset}`;
  const stagingDir = fs.mkdtempSync(path.join(os.tmpdir(), 'magpie-staging-'));

  try {
    const res = await fetchFn(downloadUrl);
    if (!res.ok) {
      throw new Error(`Failed to download ${downloadUrl}: ${res.status} ${res.statusText}`);
    }

    const arrayBuffer = await res.arrayBuffer();
    const buffer = Buffer.from(arrayBuffer);

    // Verify archive SHA-256
    const archiveHash = sha256Buffer(buffer);
    if (archiveHash !== target.sha256.toLowerCase()) {
      throw new Error(`Archive SHA256 mismatch for ${target.asset}: expected ${target.sha256}, got ${archiveHash}`);
    }

    const stagingArchive = path.join(stagingDir, target.asset);
    fs.writeFileSync(stagingArchive, buffer);

    const extractedDir = path.join(stagingDir, 'extracted');
    extractArchive(stagingArchive, extractedDir);

    const foundBinary = findBinary(extractedDir, binaryName);
    if (!foundBinary) {
      throw new Error(`Extracted archive did not contain expected binary: ${binaryName}`);
    }

    // Verify binary SHA-256
    const binaryHash = sha256File(foundBinary);
    if (binaryHash !== target.binarySha256.toLowerCase()) {
      throw new Error(`Binary SHA256 mismatch for ${binaryName}: expected ${target.binarySha256}, got ${binaryHash}`);
    }

    // Install to target binary path (handle running Windows process gracefully)
    fs.mkdirSync(packageDir, { recursive: true });
    if (platform === 'win32' && fs.existsSync(targetBinaryPath)) {
      try {
        cp.execSync(
          `powershell -NoProfile -NonInteractive -Command "Get-CimInstance Win32_Process -Filter \\"Name='${binaryName}'\\" | Where-Object { \\$_.ExecutablePath -eq '${targetBinaryPath.replace(/'/g, "''")}' } | ForEach-Object { Stop-Process -Id \\$_.ProcessId -Force -ErrorAction SilentlyContinue }"`,
          { stdio: 'ignore' }
        );
      } catch {
        // Ignore process termination errors
      }
      const backupPath = `${targetBinaryPath}.old`;
      try {
        fs.rmSync(backupPath, { force: true });
      } catch {}
      try {
        fs.renameSync(targetBinaryPath, backupPath);
        try {
          fs.rmSync(backupPath, { force: true });
        } catch {}
      } catch {
        // Fall back to direct copy if rename fails
      }
    }
    fs.copyFileSync(foundBinary, targetBinaryPath);
    if (platform !== 'win32') {
      try {
        fs.chmodSync(targetBinaryPath, 0o755);
      } catch {
        // Ignore chmod error if filesystem doesn't support it
      }
    }

    // Write install-manifest.json
    const manifestData = {
      version: pinInfo.version,
      platform,
      arch,
      archiveSha256: target.sha256.toLowerCase(),
      binarySha256: target.binarySha256.toLowerCase()
    };
    fs.writeFileSync(manifestPath, JSON.stringify(manifestData, null, 2), 'utf8');

    return {
      updated: true,
      version: pinInfo.version,
      binaryPath: targetBinaryPath
    };
  } finally {
    fs.rmSync(stagingDir, { recursive: true, force: true });
  }
}

/**
 * Standard OpenAI tools exposed to the agent.
 */
const AGENT_TOOLS = [
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
const DEFAULT_TOOL_EXECUTOR = {
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

  const systemPrompt = `You are an automated upstream convergence repair agent for CLIProxyAPI + magpie-gemini-web.
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
 * Fetch the latest studio-v* release and parse its build.json from BUKOWSKIREAL/magpie-gemini-web.
 */
export async function fetchLatestStudioRelease({
  githubToken = process.env.GITHUB_TOKEN,
  fetchFn = globalThis.fetch
} = {}) {
  const headers = {
    'User-Agent': 'ci-upstream-agent',
    'Accept': 'application/vnd.github.v3+json'
  };
  if (githubToken) {
    headers['Authorization'] = `token ${githubToken}`;
  }

  const releasesUrl = 'https://api.github.com/repos/BUKOWSKIREAL/magpie-gemini-web/releases';
  const res = await fetchFn(releasesUrl, { headers });
  if (!res.ok) {
    throw new Error(`Failed to list GitHub releases (${res.status} ${res.statusText})`);
  }

  const releases = await res.json();
  const studioRelease = releases.find(r => r.tag_name && r.tag_name.startsWith('studio-v'));
  if (!studioRelease) {
    throw new Error('No studio-v* release found in repository');
  }

  const buildAsset = studioRelease.assets?.find(a => a.name === 'build.json');
  const buildJsonUrl = buildAsset?.browser_download_url ||
    `https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/${studioRelease.tag_name}/build.json`;

  const buildRes = await fetchFn(buildJsonUrl, { headers });
  if (!buildRes.ok) {
    throw new Error(`Failed to download build.json (${buildRes.status} ${buildRes.statusText})`);
  }

  const buildJson = await buildRes.json();
  const pinInfo = parseStudioBuildJson(buildJson, studioRelease.tag_name);

  // Also check if newer regular v* release exists
  const standardRelease = releases.find(r => r.tag_name && /^v\d+\./.test(r.tag_name));

  return {
    pinInfo,
    studioRelease,
    standardRelease
  };
}

/**
 * Write key-value pairs to GITHUB_OUTPUT if set.
 */
function setGithubOutput(entries) {
  const githubOutput = process.env.GITHUB_OUTPUT;
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
  const isInstallLocal = args.includes('--install-local');
  const isSyncCi = args.includes('--sync-ci');

  if (isInstallLocal) {
    console.log('[install-local] Fetching latest MAGPIE studio release...');
    const { pinInfo } = await fetchLatestStudioRelease();
    const dataDir = process.env.AI_STUDIO_DATA_DIR || path.join(os.homedir(), '.local', 'share', 'magpie-ai-studio');
    console.log(`[install-local] Installing target binary for ${pinInfo.version} into ${dataDir}...`);
    const installResult = await installLocalStudioBinary({ dataDir, pinInfo });
    console.log(JSON.stringify(installResult, null, 2));
    return;
  }

  if (isSyncCi) {
    console.log('[sync-ci] Starting Dual-Upstream Sync Cycle...');
    const repoRoot = process.cwd();
    let changed = false;
    let verified = false;
    let summary = '';

    // Step 1: Check and update Studio Release Pin
    let pinChanged = false;
    try {
      const { pinInfo, standardRelease } = await fetchLatestStudioRelease();
      const pinUpdate = updateStudioReleasePinFiles({ repoRoot, pinInfo });
      pinChanged = pinUpdate.changed;
      if (pinChanged) {
        console.log(`[sync-ci] Advanced Studio Release Pin from ${pinUpdate.previousVersion} to ${pinUpdate.newVersion}`);
      } else {
        console.log(`[sync-ci] Studio Release Pin is already at ${pinUpdate.newVersion}`);
      }
      if (standardRelease) {
        console.log(`[sync-ci] Latest magpie-gemini-web plugin release is ${standardRelease.tag_name}`);
      }
    } catch (err) {
      console.warn(`[sync-ci] Studio release check warning: ${err.message}`);
    }

    // Step 2: Git fetch and merge upstream CLIProxyAPI main
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

    // Step 3: Run initial verification
    console.log('[sync-ci] Running initial verification gate...');
    const initialVerify = await runVerification({ cwd: repoRoot });

    if (!mergeHadConflicts && initialVerify.ok) {
      console.log('[sync-ci] All verification checks passed cleanly without repair.');
      verified = true;
      summary = 'Dual-upstream convergence verified cleanly without conflicts.';
    } else {
      console.log('[sync-ci] Verification failed or merge had conflicts. Invoking Agentic Conflict Resolution...');
      const repairContext = [
        `Merge conflicts encountered: ${mergeHadConflicts}`,
        `Verification output:\n${initialVerify.output || '(none)'}`
      ].join('\n');

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
    }

    // Step 4: Detect any uncommitted or changed files
    try {
      const gitStatus = cp.execSync('git status --porcelain', { encoding: 'utf8', cwd: repoRoot }).trim();
      changed = gitStatus.length > 0 || pinChanged || gitMerged;
    } catch {
      changed = pinChanged || gitMerged;
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
  --install-local  Download and install latest studio binary to local data dir
  --sync-ci        Execute full Dual-Upstream Sync Cycle for CI
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
