#!/usr/bin/env node

/**
 * export-magpie-bundle.mjs
 *
 * Exports MAGPIE (AIStudio2API) Google accounts and session states from a local
 * desktop environment into a portable MagpieAccountBundle (v1) JSON file for
 * importing into the Cloud Run headless gateway.
 *
 * Zero-dependency: Node.js >= 20 built-ins only.
 *
 * Usage:
 *   node scripts/export-magpie-bundle.mjs [options]
 *
 * Options:
 *   -d, --data-dir <path>  Path to magpie-ai-studio data directory
 *                          (default: ~/.local/share/magpie-ai-studio, or AI_STUDIO_DATA_DIR env)
 *   -o, --out <path>       Output JSON file path (default: magpie-accounts-bundle.json)
 *   -h, --help             Show this help message
 */

import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import process from 'node:process';

function printHelp() {
  console.log(`
MAGPIE Account State Bundle Exporter (v1)

Usage:
  node scripts/export-magpie-bundle.mjs [options]

Options:
  -d, --data-dir <path>  MAGPIE data directory containing 'auth' subfolder
                         (Default: ~/.local/share/magpie-ai-studio or AI_STUDIO_DATA_DIR env)
  -o, --out <path>       Output bundle JSON file
                         (Default: magpie-accounts-bundle.json)
  -h, --help             Display this help message

Description:
  Scans <data-dir>/auth/ for valid Google account storage states, fingerprints,
  and WAA interpreters, then packages them into an exact MagpieAccountBundle (v1)
  file ready for Cloud Run upload via the unified management console.
`);
}

function resolvePath(inputPath) {
  if (!inputPath || typeof inputPath !== 'string') return '';
  const trimmed = inputPath.trim();
  if (trimmed === '~') {
    return os.homedir();
  }
  if (trimmed.startsWith('~/' ) || trimmed.startsWith('~\\')) {
    return path.join(os.homedir(), trimmed.slice(2));
  }
  return path.resolve(trimmed);
}

function parseArgs() {
  const args = process.argv.slice(2);
  let dataDir = null;
  let outPath = null;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '-h' || arg === '--help') {
      printHelp();
      process.exit(0);
    } else if (arg === '-d' || arg === '--data-dir') {
      if (i + 1 < args.length) {
        dataDir = args[++i];
      } else {
        console.error('Error: Missing value for --data-dir');
        process.exit(1);
      }
    } else if (arg.startsWith('--data-dir=')) {
      dataDir = arg.slice('--data-dir='.length);
    } else if (arg === '-o' || arg === '--out') {
      if (i + 1 < args.length) {
        outPath = args[++i];
      } else {
        console.error('Error: Missing value for --out');
        process.exit(1);
      }
    } else if (arg.startsWith('--out=')) {
      outPath = arg.slice('--out='.length);
    } else {
      console.error(`Error: Unknown option '${arg}'. Use --help for usage.`);
      process.exit(1);
    }
  }

  // Fallback data directory
  if (!dataDir) {
    if (process.env.AI_STUDIO_DATA_DIR && process.env.AI_STUDIO_DATA_DIR.trim()) {
      dataDir = process.env.AI_STUDIO_DATA_DIR.trim();
    } else {
      dataDir = path.join(os.homedir(), '.local', 'share', 'magpie-ai-studio');
    }
  }

  // Fallback output path
  if (!outPath) {
    outPath = 'magpie-accounts-bundle.json';
  }

  return {
    dataDir: resolvePath(dataDir),
    outPath: resolvePath(outPath),
  };
}

function readJsonIfExists(filePath) {
  try {
    if (fs.existsSync(filePath) && fs.statSync(filePath).isFile()) {
      const raw = fs.readFileSync(filePath, 'utf8');
      return JSON.parse(raw);
    }
  } catch (err) {
    // Ignore unreadable/invalid optional JSON files
  }
  return undefined;
}

function exportAccountsBundle(dataDir) {
  const authDir = path.join(dataDir, 'auth');
  const accounts = [];
  const waaInterpreters = {};

  if (!fs.existsSync(dataDir)) {
    throw new Error(`Data directory does not exist: ${dataDir}`);
  }

  if (!fs.existsSync(authDir) || !fs.statSync(authDir).isDirectory()) {
    console.warn(`[WARN] 'auth' directory not found in ${dataDir}. Creating bundle with 0 accounts.`);
    return {
      version: 1,
      exported_at: new Date().toISOString(),
      accounts: [],
      waa_interpreters: waaInterpreters,
    };
  }

  // 1. Read .waa-interpreters if present
  const waaDir = path.join(authDir, '.waa-interpreters');
  if (fs.existsSync(waaDir) && fs.statSync(waaDir).isDirectory()) {
    try {
      const files = fs.readdirSync(waaDir);
      for (const file of files) {
        if (file.endsWith('.js')) {
          const filePath = path.join(waaDir, file);
          try {
            if (fs.statSync(filePath).isFile()) {
              const code = fs.readFileSync(filePath, 'utf8');
              waaInterpreters[file] = code;
            }
          } catch (err) {
            console.warn(`[WARN] Failed to read interpreter ${file}: ${err.message}`);
          }
        }
      }
    } catch (err) {
      console.warn(`[WARN] Failed to read .waa-interpreters directory: ${err.message}`);
    }
  }

  // 2. Scan account directories
  const entries = fs.readdirSync(authDir);
  for (const entry of entries) {
    // Skip hidden / metadata folders starting with '.'
    if (entry.startsWith('.')) {
      continue;
    }

    const accountDirPath = path.join(authDir, entry);
    try {
      if (!fs.statSync(accountDirPath).isDirectory()) {
        continue;
      }
    } catch {
      continue;
    }

    const storageStatePath = path.join(accountDirPath, 'storage-state.json');
    if (!fs.existsSync(storageStatePath)) {
      continue;
    }

    let storageState = null;
    try {
      const raw = fs.readFileSync(storageStatePath, 'utf8');
      const parsed = JSON.parse(raw);
      if (parsed && typeof parsed === 'object') {
        storageState = parsed;
      }
    } catch (err) {
      console.warn(`[WARN] Invalid storage-state.json in account '${entry}': ${err.message}`);
      continue;
    }

    if (!storageState) {
      continue;
    }

    const accountItem = {
      account_id: entry,
      storage_state: storageState,
    };

    const accountMeta = readJsonIfExists(path.join(accountDirPath, 'account.json'));
    if (accountMeta !== undefined) {
      accountItem.account_meta = accountMeta;
    }

    const camoufoxFingerprint = readJsonIfExists(path.join(accountDirPath, 'camoufox-fingerprint.json'));
    if (camoufoxFingerprint !== undefined) {
      accountItem.camoufox_fingerprint = camoufoxFingerprint;
    }

    const catalog = readJsonIfExists(path.join(accountDirPath, 'catalog.json'));
    if (catalog !== undefined) {
      accountItem.catalog = catalog;
    }

    const runtimeState = readJsonIfExists(path.join(accountDirPath, 'runtime-state.json'));
    if (runtimeState !== undefined) {
      accountItem.runtime_state = runtimeState;
    }

    accounts.push(accountItem);
  }

  // Sort accounts by account_id ascending
  accounts.sort((a, b) => a.account_id.localeCompare(b.account_id));

  return {
    version: 1,
    exported_at: new Date().toISOString(),
    accounts,
    waa_interpreters: waaInterpreters,
  };
}

function main() {
  const { dataDir, outPath } = parseArgs();

  console.log(`MAGPIE Account Exporter`);
  console.log(`Source data dir: ${dataDir}`);
  console.log(`Output bundle  : ${outPath}`);

  try {
    const bundle = exportAccountsBundle(dataDir);
    const accountsCount = bundle.accounts.length;
    const waaCount = Object.keys(bundle.waa_interpreters).length;

    const outDir = path.dirname(outPath);
    if (!fs.existsSync(outDir)) {
      fs.mkdirSync(outDir, { recursive: true });
    }

    const jsonStr = JSON.stringify(bundle, null, 2);
    fs.writeFileSync(outPath, jsonStr, 'utf8');

    console.log(`\nExport complete:`);
    console.log(`- Accounts packaged    : ${accountsCount}`);
    console.log(`- WAA interpreters     : ${waaCount}`);
    console.log(`- Bundle size          : ${(Buffer.byteLength(jsonStr, 'utf8') / 1024).toFixed(1)} KB`);
    console.log(`- File saved to        : ${outPath}`);
    console.log(`\nNext step:`);
    console.log(`Upload '${path.basename(outPath)}' in the Cloud Run management console under MAGPIE 配额 -> 导入账号包.`);
  } catch (err) {
    console.error(`\n[ERROR] Export failed: ${err.message}`);
    process.exit(1);
  }
}

main();
