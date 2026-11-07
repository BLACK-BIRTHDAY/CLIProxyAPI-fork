import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import cp from 'node:child_process';

import {
  parseStudioBuildJson,
  updateStudioReleasePinFiles,
  installLocalStudioBinary,
  runAgenticRepairLoop,
  sha256File
} from './ci-upstream-agent.mjs';

const SAMPLE_BUILD_JSON = {
  version: '0.3.0-magpie.1',
  binaries: {
    'darwin-arm64': {
      asset: 'aistudio2api-v0.3.0-magpie.1-darwin-arm64.tar.gz',
      sha256: 'fa2be9428645977c5fd1f88b373f2531d81f86dc0da1cb367918639ec2cc6ea0',
      binarySha256: 'ef368372a3c6d9b90a9ac50743b9e01526fb8370355606b2a47117f6d22071e0'
    },
    'darwin-x64': {
      asset: 'aistudio2api-v0.3.0-magpie.1-darwin-amd64.tar.gz',
      sha256: '2865c869800a211e57dd8362e5a0cad95b79c14228c03ed56e63477dc99317d2',
      binarySha256: '85d59752c453c77d6d6c8289f50627029ead22d429c0064d3545d198d90d8963'
    },
    'linux-x64': {
      asset: 'aistudio2api-v0.3.0-magpie.1-linux-amd64.tar.gz',
      sha256: '432b4e7650f2d86f82965bdec221522a4d7e0ad2d49e1c58c3a3f0cb7fc1304b',
      binarySha256: 'cbce67249646dfe4006ec9f7e50ac141136755b0d26a04c435700ca352dd0129'
    },
    'linux-arm64': {
      asset: 'aistudio2api-v0.3.0-magpie.1-linux-arm64.tar.gz',
      sha256: '5eb213627e48384d3e50c26bc1f056ddcd9eedce617847b1e8d477c529ecc5f8',
      binarySha256: '10b8b30de5412f5855fabd314f25fa01a539d464a683b1b6e78608a549aac2cf'
    },
    'win32-x64': {
      asset: 'aistudio2api-v0.3.0-magpie.1-windows-amd64.zip',
      sha256: 'f426a010fb40505acb1499de523328a7afb386f551ab68781b4c124dbc2c9ba2',
      binarySha256: 'cb3cd70312f2df0e2f400621abc268dde7ba780979ccd7e95f2acebb8e9349cc'
    }
  }
};

const SAMPLE_DOCKERFILE = `FROM golang:1.26-bookworm AS builder
WORKDIR /app
# Pre-install pinned MAGPIE AIStudio2API binary
ARG TARGETARCH
RUN set -e; \\
    ARCH="\${TARGETARCH:-\$(dpkg --print-architecture)}"; \\
    case "\$ARCH" in \\
        amd64|x86_64) \\
            URL="https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/studio-v0.2.19-magpie.1/aistudio2api-v0.2.19-magpie.1-linux-amd64.tar.gz"; \\
            ARCHIVE_SHA256="8dfaf0d4d2a263ca3f4d64ba6f82841331d4a5c937724ab8b33bb6a61af4c6c5"; \\
            BIN_SHA256="384b092cb05172b49666b7e3be39bcce377e7b2aff33f94c3287f280ce72c877"; \\
            ;; \\
        arm64|aarch64) \\
            URL="https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/studio-v0.2.19-magpie.1/aistudio2api-v0.2.19-magpie.1-linux-arm64.tar.gz"; \\
            ARCHIVE_SHA256="4c20174a731ea699c2a0adab686f063d88088ff93cbd33c399f444fc4b5bcbaa"; \\
            BIN_SHA256="c82303992bd340883a69fada20a34eb8491ea9f653ed966df4ed98efe6a81470"; \\
            ;; \\
        *) \\
            echo "Unsupported architecture: \$ARCH" >&2; \\
            exit 1; \\
            ;; \\
    esac;
`;

const SAMPLE_PROCESS_GO = `package magpie

const (
	// MagpieStudioVersion is the pinned upstream AIStudio2API release version.
	MagpieStudioVersion = "0.2.19-magpie.1"
	// PreinstalledMagpieBin is the container-bundled binary path for Cloud Run.
	PreinstalledMagpieBin = "/app/magpie/aistudio2api"
)
`;

test('parseStudioBuildJson: valid build.json extraction and validation', () => {
  const pinInfo = parseStudioBuildJson(SAMPLE_BUILD_JSON, 'studio-v0.3.0-magpie.1');
  assert.equal(pinInfo.version, '0.3.0-magpie.1');
  assert.equal(pinInfo.tag, 'studio-v0.3.0-magpie.1');
  assert.equal(Object.keys(pinInfo.binaries).length, 5);
  assert.equal(pinInfo.binaries['linux-x64'].asset, 'aistudio2api-v0.3.0-magpie.1-linux-amd64.tar.gz');
  assert.equal(pinInfo.binaries['linux-x64'].sha256, '432b4e7650f2d86f82965bdec221522a4d7e0ad2d49e1c58c3a3f0cb7fc1304b');
  assert.equal(pinInfo.binaries['linux-x64'].binarySha256, 'cbce67249646dfe4006ec9f7e50ac141136755b0d26a04c435700ca352dd0129');
});

test('parseStudioBuildJson: parses JSON string and infers default tag', () => {
  const jsonStr = JSON.stringify(SAMPLE_BUILD_JSON);
  const pinInfo = parseStudioBuildJson(jsonStr);
  assert.equal(pinInfo.version, '0.3.0-magpie.1');
  assert.equal(pinInfo.tag, 'studio-v0.3.0-magpie.1');
});

test('parseStudioBuildJson: rejects missing version or invalid digest', () => {
  assert.throws(
    () => parseStudioBuildJson({ binaries: SAMPLE_BUILD_JSON.binaries }),
    /missing or invalid "version"/
  );

  assert.throws(
    () => parseStudioBuildJson({ version: '1.0.0' }),
    /missing "binaries" object/
  );

  const invalidSha = JSON.parse(JSON.stringify(SAMPLE_BUILD_JSON));
  invalidSha.binaries['linux-x64'].sha256 = 'not-a-valid-sha256';
  assert.throws(
    () => parseStudioBuildJson(invalidSha),
    /missing or invalid 64-hex "sha256"/
  );

  const invalidBinSha = JSON.parse(JSON.stringify(SAMPLE_BUILD_JSON));
  invalidBinSha.binaries['linux-x64'].binarySha256 = 'short-hash';
  assert.throws(
    () => parseStudioBuildJson(invalidBinSha),
    /missing or invalid 64-hex "binarySha256"/
  );

  const missingAsset = JSON.parse(JSON.stringify(SAMPLE_BUILD_JSON));
  delete missingAsset.binaries['linux-x64'].asset;
  assert.throws(
    () => parseStudioBuildJson(missingAsset),
    /missing "asset"/
  );

  const missingTarget = JSON.parse(JSON.stringify(SAMPLE_BUILD_JSON));
  delete missingTarget.binaries['darwin-x64'];
  assert.throws(
    () => parseStudioBuildJson(missingTarget),
    /missing target "darwin-x64"/
  );
});

test('updateStudioReleasePinFiles: updates Dockerfile and process.go deterministically', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pin-test-'));
  try {
    const dockerfilePath = path.join(tempDir, 'Dockerfile');
    const processGoDir = path.join(tempDir, 'internal', 'magpie');
    fs.mkdirSync(processGoDir, { recursive: true });
    const processGoPath = path.join(processGoDir, 'process.go');

    fs.writeFileSync(dockerfilePath, SAMPLE_DOCKERFILE, 'utf8');
    fs.writeFileSync(processGoPath, SAMPLE_PROCESS_GO, 'utf8');

    const pinInfo = parseStudioBuildJson(SAMPLE_BUILD_JSON);
    const res1 = updateStudioReleasePinFiles({ repoRoot: tempDir, pinInfo });

    assert.equal(res1.changed, true);
    assert.equal(res1.previousVersion, '0.2.19-magpie.1');
    assert.equal(res1.newVersion, '0.3.0-magpie.1');

    // Verify written content
    const updatedDockerfile = fs.readFileSync(dockerfilePath, 'utf8');
    assert.match(updatedDockerfile, /studio-v0\.3\.0-magpie\.1\/aistudio2api-v0\.3\.0-magpie\.1-linux-amd64\.tar\.gz/);
    assert.match(updatedDockerfile, /432b4e7650f2d86f82965bdec221522a4d7e0ad2d49e1c58c3a3f0cb7fc1304b/);
    assert.match(updatedDockerfile, /cbce67249646dfe4006ec9f7e50ac141136755b0d26a04c435700ca352dd0129/);
    assert.match(updatedDockerfile, /studio-v0\.3\.0-magpie\.1\/aistudio2api-v0\.3\.0-magpie\.1-linux-arm64\.tar\.gz/);
    assert.match(updatedDockerfile, /5eb213627e48384d3e50c26bc1f056ddcd9eedce617847b1e8d477c529ecc5f8/);
    assert.match(updatedDockerfile, /10b8b30de5412f5855fabd314f25fa01a539d464a683b1b6e78608a549aac2cf/);

    const updatedProcessGo = fs.readFileSync(processGoPath, 'utf8');
    assert.match(updatedProcessGo, /MagpieStudioVersion = "0\.3\.0-magpie\.1"/);

    // Second call is idempotent: changed = false
    const res2 = updateStudioReleasePinFiles({ repoRoot: tempDir, pinInfo });
    assert.equal(res2.changed, false);
    assert.equal(res2.previousVersion, '0.3.0-magpie.1');
    assert.equal(res2.newVersion, '0.3.0-magpie.1');
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test('updateStudioReleasePinFiles: throws when files are missing', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pin-missing-test-'));
  try {
    const pinInfo = parseStudioBuildJson(SAMPLE_BUILD_JSON);
    assert.throws(
      () => updateStudioReleasePinFiles({ repoRoot: tempDir, pinInfo }),
      /Dockerfile not found/
    );
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test('installLocalStudioBinary: fast return when already matching manifest and binary', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'install-test-'));
  try {
    const packageDir = path.join(tempDir, 'package');
    fs.mkdirSync(packageDir, { recursive: true });
    const binaryName = process.platform === 'win32' ? 'aistudio2api.exe' : 'aistudio2api';
    const binaryPath = path.join(packageDir, binaryName);
    fs.writeFileSync(binaryPath, 'mock binary content');

    const pinInfo = parseStudioBuildJson(SAMPLE_BUILD_JSON);
    const targetKey = process.platform === 'win32' ? 'win32-x64' : 'linux-x64';
    const target = pinInfo.binaries[targetKey];

    const manifest = {
      version: pinInfo.version,
      platform: process.platform,
      arch: 'x64',
      archiveSha256: target.sha256,
      binarySha256: target.binarySha256
    };
    fs.writeFileSync(path.join(tempDir, 'install-manifest.json'), JSON.stringify(manifest), 'utf8');

    let fetchCalled = false;
    const fetchFn = async () => {
      fetchCalled = true;
      throw new Error('should not fetch');
    };

    const res = await installLocalStudioBinary({
      dataDir: tempDir,
      platform: process.platform,
      arch: 'x64',
      pinInfo,
      fetchFn
    });

    assert.equal(res.updated, false);
    assert.equal(res.version, pinInfo.version);
    assert.equal(res.binaryPath, binaryPath);
    assert.equal(fetchCalled, false);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test('installLocalStudioBinary: downloads, verifies sha256, extracts, and installs', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'install-download-test-'));
  const stagingTestDir = fs.mkdtempSync(path.join(os.tmpdir(), 'staging-test-'));
  try {
    const binaryName = process.platform === 'win32' ? 'aistudio2api.exe' : 'aistudio2api';
    const mockBinFile = path.join(stagingTestDir, binaryName);
    const mockBinContent = 'actual mock binary bytes 1234567890';
    fs.writeFileSync(mockBinFile, mockBinContent);
    const actualBinarySha256 = crypto.createHash('sha256').update(mockBinContent).digest('hex');

    // Create tar.gz archive
    const archivePath = path.join(stagingTestDir, 'mock-archive.tar.gz');
    const tarBin = process.platform === 'win32' && fs.existsSync('C:/Windows/System32/tar.exe')
      ? 'C:/Windows/System32/tar.exe'
      : 'tar';
    cp.execFileSync(tarBin, ['-czf', archivePath, '-C', stagingTestDir, binaryName]);

    const archiveBuffer = fs.readFileSync(archivePath);
    const actualArchiveSha256 = crypto.createHash('sha256').update(archiveBuffer).digest('hex');

    const customPinInfo = {
      version: '0.9.9-test',
      tag: 'studio-v0.9.9-test',
      binaries: {
        'linux-x64': {
          asset: 'mock-archive.tar.gz',
          sha256: actualArchiveSha256,
          binarySha256: actualBinarySha256
        },
        'win32-x64': {
          asset: 'mock-archive.tar.gz',
          sha256: actualArchiveSha256,
          binarySha256: actualBinarySha256
        }
      }
    };

    const mockFetch = async () => ({
      ok: true,
      status: 200,
      statusText: 'OK',
      arrayBuffer: async () => archiveBuffer.buffer.slice(archiveBuffer.byteOffset, archiveBuffer.byteOffset + archiveBuffer.byteLength)
    });

    const res = await installLocalStudioBinary({
      dataDir: tempDir,
      platform: process.platform === 'win32' ? 'win32' : 'linux',
      arch: 'x64',
      pinInfo: customPinInfo,
      fetchFn: mockFetch
    });

    assert.equal(res.updated, true);
    assert.equal(res.version, '0.9.9-test');
    assert.equal(fs.existsSync(res.binaryPath), true);
    assert.equal(fs.readFileSync(res.binaryPath, 'utf8'), mockBinContent);

    // Verify manifest
    const manifestPath = path.join(tempDir, 'install-manifest.json');
    assert.equal(fs.existsSync(manifestPath), true);
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    assert.equal(manifest.version, '0.9.9-test');
    assert.equal(manifest.binarySha256, actualBinarySha256);
    assert.equal(manifest.archiveSha256, actualArchiveSha256);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
    fs.rmSync(stagingTestDir, { recursive: true, force: true });
  }
});

test('installLocalStudioBinary: throws on binary SHA256 mismatch after extraction', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'install-bin-mismatch-test-'));
  const stagingTestDir = fs.mkdtempSync(path.join(os.tmpdir(), 'staging-bin-mismatch-'));
  try {
    const binaryName = process.platform === 'win32' ? 'aistudio2api.exe' : 'aistudio2api';
    const mockBinFile = path.join(stagingTestDir, binaryName);
    fs.writeFileSync(mockBinFile, 'binary content');

    const archivePath = path.join(stagingTestDir, 'mock-archive.tar.gz');
    const tarBin = process.platform === 'win32' && fs.existsSync('C:/Windows/System32/tar.exe')
      ? 'C:/Windows/System32/tar.exe'
      : 'tar';
    cp.execFileSync(tarBin, ['-czf', archivePath, '-C', stagingTestDir, binaryName]);

    const archiveBuffer = fs.readFileSync(archivePath);
    const actualArchiveSha256 = crypto.createHash('sha256').update(archiveBuffer).digest('hex');

    const customPinInfo = {
      version: '0.9.9-test',
      tag: 'studio-v0.9.9-test',
      binaries: {
        'win32-x64': {
          asset: 'mock-archive.tar.gz',
          sha256: actualArchiveSha256,
          binarySha256: 'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
        },
        'linux-x64': {
          asset: 'mock-archive.tar.gz',
          sha256: actualArchiveSha256,
          binarySha256: 'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
        }
      }
    };

    const mockFetch = async () => ({
      ok: true,
      status: 200,
      arrayBuffer: async () => archiveBuffer.buffer.slice(archiveBuffer.byteOffset, archiveBuffer.byteOffset + archiveBuffer.byteLength)
    });

    await assert.rejects(
      async () => installLocalStudioBinary({
        dataDir: tempDir,
        platform: process.platform === 'win32' ? 'win32' : 'linux',
        arch: 'x64',
        pinInfo: customPinInfo,
        fetchFn: mockFetch
      }),
      /Binary SHA256 mismatch/
    );
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
    fs.rmSync(stagingTestDir, { recursive: true, force: true });
  }
});

test('installLocalStudioBinary: throws on unsupported platform', async () => {
  const pinInfo = parseStudioBuildJson(SAMPLE_BUILD_JSON);
  await assert.rejects(
    async () => installLocalStudioBinary({
      dataDir: os.tmpdir(),
      platform: 'solaris',
      arch: 'sparc',
      pinInfo
    }),
    /Unsupported platform\/architecture/
  );
});

test('installLocalStudioBinary: throws on archive SHA256 mismatch', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'install-err-test-'));
  try {
    const customPinInfo = {
      version: '0.9.9-test',
      tag: 'studio-v0.9.9-test',
      binaries: {
        'win32-x64': {
          asset: 'archive.zip',
          sha256: '0000000000000000000000000000000000000000000000000000000000000000',
          binarySha256: '0000000000000000000000000000000000000000000000000000000000000000'
        },
        'linux-x64': {
          asset: 'archive.tar.gz',
          sha256: '0000000000000000000000000000000000000000000000000000000000000000',
          binarySha256: '0000000000000000000000000000000000000000000000000000000000000000'
        }
      }
    };

    const mockFetch = async () => ({
      ok: true,
      status: 200,
      arrayBuffer: async () => Buffer.from('bad data')
    });

    await assert.rejects(
      async () => installLocalStudioBinary({
        dataDir: tempDir,
        platform: process.platform === 'win32' ? 'win32' : 'linux',
        arch: 'x64',
        pinInfo: customPinInfo,
        fetchFn: mockFetch
      }),
      /Archive SHA256 mismatch/
    );
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
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
