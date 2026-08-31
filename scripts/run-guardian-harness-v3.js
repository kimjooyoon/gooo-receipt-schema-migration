'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const vm = require('node:vm');
const {createRequire} = require('node:module');

function parseArgs(argv) {
  const values = {};
  for (let index = 0; index < argv.length; index += 1) {
    if (!argv[index].startsWith('--') || index + 1 >= argv.length) throw new Error(`invalid argument ${argv[index] || ''}`);
    values[argv[index].slice(2)] = argv[index + 1];
    index += 1;
  }
  return values;
}

function readJSON(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function writeJSON(file, value) {
  fs.mkdirSync(path.dirname(file), {recursive: true});
  fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
}

function gitBlobSHA1(bytes) {
  const header = Buffer.from(`blob ${bytes.length}\0`, 'utf8');
  return crypto.createHash('sha1').update(Buffer.concat([header, bytes])).digest('hex');
}

function sha256String(value) {
  return `sha256:${crypto.createHash('sha256').update(value, 'utf8').digest('hex')}`;
}

function changedPathsDigest(files) {
  const paths = [...new Set(files.flatMap((file) => [file.filename, file.previous_filename]).filter(Boolean))].sort();
  return sha256String(paths.length === 0 ? '' : `${paths.join('\n')}\n`);
}

function kernelEntriesDigest(entries) {
  const canonical = entries.map((entry) => ({path: entry.path, type: entry.type, mode: entry.mode || null, sha: entry.sha}))
    .sort((left, right) => {
      const leftKey = [left.path, left.type, left.sha].join('\0');
      const rightKey = [right.path, right.type, right.sha].join('\0');
      return leftKey < rightKey ? -1 : leftKey > rightKey ? 1 : 0;
    });
  return sha256String(JSON.stringify(canonical));
}

function encodeFile(bytes) {
  return {type: 'file', content: Buffer.from(bytes).toString('base64')};
}

async function fetchFixture(repoRoot, fixture) {
  const manifest = readJSON(path.join(repoRoot, fixture.manifest_path));
  if (manifest.schema !== 'gooo/receipt-schema-migration/guardian-fixture/v3' || manifest.repository !== fixture.repository || manifest.ref !== fixture.ref || manifest.base_commit !== fixture.base_commit || manifest.head_commit !== fixture.head_commit) {
    throw new Error('v3 Guardian fixture manifest is not bound to the generated fixture declaration');
  }
  if (!Array.isArray(manifest.pinned_files) || manifest.pinned_files.length === 0 || !Array.isArray(manifest.changed_files) || manifest.changed_files.length !== fixture.changed_files_count || !Array.isArray(manifest.protected_intersection) || manifest.protected_intersection.length !== fixture.protected_intersection_count || !Array.isArray(manifest.kernel_before_entries) || !Array.isArray(manifest.kernel_after_entries)) {
    throw new Error('v3 Guardian fixture manifest is incomplete');
  }
  if (changedPathsDigest(manifest.changed_files) !== fixture.changed_paths_sha256 || sha256String(`${[...manifest.protected_intersection].sort().join('\n')}\n`) !== fixture.protected_intersection_sha256 || kernelEntriesDigest(manifest.kernel_before_entries) !== fixture.kernel_before_sha256 || kernelEntriesDigest(manifest.kernel_after_entries) !== fixture.kernel_after_sha256 || manifest.kernel_before_entries.length !== fixture.kernel_before_entry_count || manifest.kernel_after_entries.length !== fixture.kernel_after_entry_count || manifest.kernel_before_tree_sha !== fixture.kernel_before_tree_sha || manifest.kernel_after_tree_sha !== fixture.kernel_after_tree_sha || manifest.guardian_run.artifact_sha256 !== fixture.artifact_sha256 || manifest.guardian_run.artifact_size_bytes !== fixture.artifact_size_bytes) {
    throw new Error('v3 Guardian fixture digest or cardinality binding is not exact');
  }
  const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'gooo-guardian-v3-'));
  for (const item of manifest.pinned_files) {
    const response = await fetch(item.source);
    if (!response.ok) throw new Error(`fixture fetch failed for ${item.path}: ${response.status}`);
    const bytes = Buffer.from(await response.arrayBuffer());
    if (gitBlobSHA1(bytes) !== item.blob_sha) throw new Error(`fixture blob SHA mismatch for ${item.path}`);
    const target = path.join(workspace, item.path);
    fs.mkdirSync(path.dirname(target), {recursive: true});
    fs.writeFileSync(target, bytes);
  }
  return {workspace, manifest};
}

function extractFeatureScript(workflow) {
  const marker = 'const eventPull = context.payload.pull_request;';
  const markerOffset = workflow.indexOf(marker);
  if (markerOffset < 0) throw new Error('feature pull marker is missing from pinned ci-guardian.yml');
  const scriptOffset = workflow.lastIndexOf('script: |', markerOffset);
  if (scriptOffset < 0) throw new Error('feature script block is missing from pinned ci-guardian.yml');
  const bodyStart = workflow.indexOf('\n', scriptOffset) + 1;
  const blockEnd = workflow.indexOf('\n      - name:', markerOffset);
  if (bodyStart <= 0 || blockEnd < bodyStart) throw new Error('feature script block boundary is malformed');
  const lines = workflow.slice(bodyStart, blockEnd).split('\n');
  const body = [];
  for (const line of lines) {
    if (line === '') body.push('');
    else if (line.startsWith('            ')) body.push(line.slice(12));
    else break;
  }
  const script = body.join('\n').replace(/\n+$/, '');
  const declarationOffset = script.indexOf('let beforeDigest = null;');
  const policyBranchOffset = script.indexOf("path: '.github/foundation-authorization.json'");
  if (!script.includes(marker) || declarationOffset < 0 || policyBranchOffset < 0 || declarationOffset > policyBranchOffset || !script.includes('computedBeforeDigest: beforeDigest')) {
    throw new Error('pinned feature script does not initialize digest scope before Foundation authorization');
  }
  return script;
}

function makePull(fixture, files) {
  return {
    number: 609,
    state: 'open',
    draft: false,
    merged: false,
    merged_at: null,
    changed_files: files.length,
    base: {ref: 'dev', sha: fixture.base_commit, repo: {full_name: fixture.repository}},
    head: {ref: 'agent/dev-main-sync-20260831-rerun', sha: fixture.head_commit, repo: {full_name: fixture.repository}},
  };
}

function changedPathFiles(manifest, mode) {
  const files = manifest.changed_files.map((file) => ({filename: file.filename, previous_filename: file.previous_filename || null, status: file.status}));
  if (mode === 'CHANGED_PATH_TUPLE_MISMATCH') {
    files[0] = {...files[0], filename: `${files[0].filename}.tuple-mismatch`};
  }
  if (mode === 'EXTRA_PROTECTED_PATH') files.push({filename: 'go.sum', previous_filename: null, status: 'modified'});
  return files;
}

function makeValidPolicy(fixture, manifest, authorization) {
  const auth = authorization;
  const manifestBytes = Buffer.from('guardian-v3-foundation-manifest\n', 'utf8');
  const candidateTreeEntries = manifest.kernel_after_entries;
  return {
    schema: auth.AUTHORIZATION_SCHEMA,
    decision: 'FOUNDATION',
    reason: 'NO_AUTHORIZED_MAIN_TO_DEV_PROTECTED_KERNEL_ROUTE',
    repository: fixture.repository,
    foundation_override_success_count: auth.FOUNDATION_OVERRIDE_SUCCESS_COUNT,
    foundation_override_marker: auth.FOUNDATION_OVERRIDE_MARKER,
    candidate: {
      pull_request: 609,
      branch: auth.CANDIDATE_BRANCH,
      base_branch: 'dev',
      base_sha: fixture.base_commit,
      head_sha: fixture.head_commit,
      manifest_path: fixture.manifest_path,
      manifest_sha256: auth.sha256(manifestBytes),
      changed_path_count: fixture.changed_files_count,
      changed_paths_sha256: fixture.changed_paths_sha256,
      patch_sha256_excluding_authorization_paths: auth.sha256(Buffer.from('guardian-v3-patch\n', 'utf8')),
      tree_sha256_excluding_authorization_paths: auth.digestTreeEntries(candidateTreeEntries, auth.AUTHORIZATION_PATHS),
    },
    authorization: {
      pull_request: 610,
      branch: auth.AUTHORIZATION_BRANCH,
      base_branch: 'dev',
      changed_paths: auth.AUTHORIZATION_PATHS,
      mode: 'single-use',
      consumed: false,
      replay_decision: 'REFUTED',
    },
    auth_policy_paths: auth.AUTHORIZATION_PATHS,
    base_attestation: {commit: fixture.base_commit, tree_sha: fixture.kernel_before_tree_sha, parents: manifest.base_parents},
    _manifest_bytes: manifestBytes,
  };
}

function policyBytes(policy) {
  const copy = {...policy};
  delete copy._manifest_bytes;
  return Buffer.from(JSON.stringify(copy), 'utf8');
}

function mockGitHub({fixture, manifest, pull, changedFiles, policyMode, guardianAuth}) {
  const mainSHA = 'a0962dd1ac8376b9e88bb629c66e3a7f710b96a9';
  const validPolicy = makeValidPolicy(fixture, manifest, guardianAuth);
  let baseCommitCalls = 0;
  let policy = validPolicy;
  if (policyMode === 'STALE_FOUNDATION_AUTHORIZATION') policy = {...validPolicy, candidate: {...validPolicy.candidate, branch: 'agent/stale-foundation-authorization'}};
  if (policyMode === 'FOUNDATION_CARDINALITY_EXHAUSTED') policy = {...validPolicy, authorization: {...validPolicy.authorization, consumed: true}};
  if (policyMode === 'FOUNDATION_REPLAY') policy = {...validPolicy, authorization: {...validPolicy.authorization, replay_decision: 'CLOSED'}};
  return {
    rest: {
      git: {
        getRef: async ({ref}) => ({status: 200, data: {object: {sha: ref === 'heads/main' ? mainSHA : fixture.base_commit}}}),
        getTree: async ({tree_sha}) => {
          if (tree_sha === fixture.kernel_before_tree_sha) return {status: 200, data: {sha: tree_sha, truncated: false, tree: manifest.kernel_before_entries}};
          if (tree_sha === fixture.kernel_after_tree_sha) return {status: 200, data: {sha: tree_sha, truncated: false, tree: manifest.kernel_after_entries}};
          return {status: 200, data: {sha: tree_sha, truncated: false, tree: []}};
        },
      },
      repos: {
        compareCommits: async ({base, head}) => {
          if (base === mainSHA && head === fixture.base_commit) return {status: 200, data: {status: 'diverged', ahead_by: 199, behind_by: 5, merge_base_commit: {sha: fixture.merge_base}}};
          if (base === fixture.head_commit && head === pull.head.sha) return {status: 200, data: {status: 'ahead', ahead_by: 1, behind_by: 0, merge_base_commit: {sha: fixture.head_commit}}};
          if (base === fixture.base_commit && head === pull.head.sha) return {status: 200, data: {status: 'ahead', ahead_by: 1, behind_by: 0, merge_base_commit: {sha: fixture.base_commit}, files: changedFiles}};
          return {status: 200, data: {status: 'ahead', ahead_by: 1, behind_by: 0, merge_base_commit: {sha: base}}};
        },
        getCommit: async ({ref}) => {
          if (ref === fixture.base_commit) {
            baseCommitCalls += 1;
            const parents = baseCommitCalls === 1 ? [{sha: fixture.base_commit}] : manifest.base_parents.map((sha) => ({sha}));
            return {status: 200, data: {sha: fixture.base_commit, parents, commit: {tree: {sha: fixture.kernel_before_tree_sha}}}};
          }
          if (ref === fixture.head_commit) return {status: 200, data: {sha: fixture.head_commit, parents: [], commit: {tree: {sha: fixture.kernel_after_tree_sha}}}};
          return {status: 200, data: {sha: ref, parents: [{sha: fixture.base_commit}], commit: {tree: {sha: fixture.kernel_before_tree_sha}}}};
        },
        getContent: async ({path: contentPath}) => {
          if (contentPath === '.github/foundation-authorization.json') {
            if (policyMode === 'MISSING_FOUNDATION_AUTHORIZATION') return {status: 404, data: null};
            if (policyMode === 'MALFORMED_FOUNDATION_AUTHORIZATION') return {status: 200, data: encodeFile(Buffer.from('{malformed', 'utf8'))};
            return {status: 200, data: encodeFile(policyBytes(policy))};
          }
          if (contentPath === fixture.manifest_path) return {status: 200, data: encodeFile(validPolicy._manifest_bytes)};
          return {status: 404, data: null};
        },
      },
      pulls: {
        get: async ({pull_number}) => {
          if (pull_number === 610) return {status: 200, data: {number: 610, state: 'closed', merged: true, merge_commit_sha: fixture.base_commit, base: {ref: 'dev', repo: {full_name: fixture.repository}}, head: {ref: guardianAuth.AUTHORIZATION_BRANCH, repo: {full_name: fixture.repository,}, sha: 'b'.repeat(40)}}};
          return {status: 200, data: pull};
        },
        listFiles: async () => ({status: 200, data: changedFiles}),
      },
    },
  };
}

async function executeWorkflowScript(script, workspace, pull, fixture, manifest, changedFiles, policyMode, guardianAuth) {
  const scriptPath = path.join(workspace, 'workflow-script-v3.js');
  fs.writeFileSync(scriptPath, script);
  const failures = [];
  const summary = {addHeading() { return this; }, addRaw() { return this; }, async write() { return undefined; }};
  const core = {summary, setFailed(message) { failures.push(String(message)); }, setOutput() {}};
  const context = {
    payload: {action: 'reopened', pull_request: pull, repository: {full_name: fixture.repository, default_branch: 'dev'}},
    repo: {owner: 'kimjooyoon', repo: 'meta-ontology-go'},
  };
  const github = mockGitHub({fixture, manifest, pull, changedFiles, policyMode, guardianAuth});
  const sandbox = {
    Buffer,
    console,
    context,
    core,
    github,
    process: {env: {
      GUARDIAN_WORKFLOW_REF: `${fixture.repository}/.github/workflows/ci-guardian.yml@refs/heads/dev`,
      GUARDIAN_WORKFLOW_SHA: fixture.base_commit,
      GUARDIAN_RUNTIME_REF: 'refs/heads/dev',
      GUARDIAN_RUNTIME_SHA: fixture.base_commit,
      GUARDIAN_RUN_ID: String(fixture.guardian_run_id),
      GUARDIAN_RUN_ATTEMPT: '1',
      GUARDIAN_APP_TOKEN: '',
      GUARDIAN_APP_INSTALLATION_ID: '0',
      GUARDIAN_APP_SLUG: '',
    }},
    require: createRequire(scriptPath),
  };
  process.chdir(workspace);
  try {
    await vm.runInNewContext(`(async () => {\n${script}\n})()`, sandbox, {filename: scriptPath});
  } catch (error) {
    return {error, failures, artifact: readOptionalJSON(path.join(workspace, 'ci-guardian.json'))};
  }
  return {error: null, failures, artifact: readOptionalJSON(path.join(workspace, 'ci-guardian.json'))};
}

function readOptionalJSON(file) {
  try { return readJSON(file); } catch { return null; }
}

function sortedObject(value) {
  return Object.fromEntries(Object.keys(value).sort().map((key) => [key, value[key]]));
}

function claim(state, stage, step, reason, unknownClass, nextOperation, blockedBy) {
  return {state, stage, step, reason, unknown_class: unknownClass, next_operation: nextOperation, blocked_by: blockedBy};
}

function resultFor(item, state, guardianDecision, value, evidence = {}) {
  const output = {
    id: item.id,
    scenario_id: item.scenario_id,
    cell_id: item.cell,
    expected_state: item.expected,
    state,
    guardian_decision: guardianDecision,
    stage: value.stage,
    step: value.step,
    reason: value.reason,
    unknown_class: value.unknownClass || '',
    next_operation: value.nextOperation,
    blocked_by: value.blockedBy || [],
    claim: claim(state, value.stage, value.step, value.reason, value.unknownClass || '', value.nextOperation, value.blockedBy || []),
  };
  if (value.beforeDigest) output.before_digest = value.beforeDigest;
  if (value.afterDigest) output.after_digest = value.afterDigest;
  if (value.artifactBeforeDigest) output.artifact_before_digest = value.artifactBeforeDigest;
  if (value.artifactAfterDigest) output.artifact_after_digest = value.artifactAfterDigest;
  if (Object.keys(evidence).length > 0) output.evidence = sortedObject(evidence);
  return output;
}

function workflowResult(item, outcome, expected, foundationEvaluated = true) {
  if (outcome.error || !outcome.artifact) throw new Error(`${item.id} did not emit a Guardian artifact: ${outcome.error ? outcome.error.message : 'missing artifact'}`);
  const passed = outcome.artifact.decision === 'PASS';
  if ((expected === 'CLOSED') !== passed) throw new Error(`${item.id} expected ${expected} but artifact decision was ${outcome.artifact.decision} (${outcome.artifact.code || 'none'}): ${outcome.artifact.reason}; foundation=${JSON.stringify(outcome.artifact.foundation_authorization || null)}`);
  if (expected === 'CLOSED' && outcome.failures.length !== 0) throw new Error(`${item.id} unexpectedly failed: ${outcome.failures.join(';')}`);
  const state = expected === 'CLOSED' ? 'CLOSED' : 'REFUTED';
  return resultFor(item, state, passed ? 'PASS' : 'REFUTED', {
    stage: 'GUARDIAN',
    step: passed ? 'validate_kernel_digest_attestation' : 'fail_closed_foundation_authorization',
    reason: outcome.artifact.reason,
    nextOperation: passed ? 'NONE' : 'REPAIR_FOUNDATION_AUTHORIZATION_EVIDENCE',
    blockedBy: passed ? [] : ['foundation-authorization', 'changed-path-tuples'],
    beforeDigest: outcome.artifact.kernel_before_sha256,
    afterDigest: outcome.artifact.kernel_after_sha256,
    artifactBeforeDigest: outcome.artifact.kernel_before_sha256,
    artifactAfterDigest: outcome.artifact.kernel_after_sha256,
  }, {
    artifact_decision: outcome.artifact.decision,
    artifact_code: outcome.artifact.code,
    changed_files_count: outcome.artifact.changed_files_count,
    protected_intersection_count: outcome.artifact.kernel_paths.length,
    foundation_authorization_evaluated: foundationEvaluated,
    foundation_receipt_evaluated: foundationEvaluated,
  });
}

function digestResult(guardian, item, fixture, mode) {
  const before = fixture.kernel_before_sha256;
  const after = fixture.kernel_after_sha256;
  let computedBefore = before;
  let computedAfter = after;
  let artifactBefore = before;
  let artifactAfter = after;
  if (mode === 'PASS_NULL_KERNEL_DIGEST') {
    computedBefore = null;
    computedAfter = null;
    artifactBefore = null;
    artifactAfter = null;
  }
  if (mode === 'PASS_MISMATCH_KERNEL_DIGEST') artifactAfter = `sha256:${'c'.repeat(64)}`;
  const attestation = guardian.validateKernelDigestAttestation({
    kernelPaths: ['scripts/ci-proof/guardian.js'],
    computedBeforeDigest: computedBefore,
    computedAfterDigest: computedAfter,
    artifactBeforeDigest: artifactBefore,
    artifactAfterDigest: artifactAfter,
  });
  if (attestation.decision !== 'REFUTED') throw new Error(`${item.id} did not refute kernel digest evidence`);
  return resultFor(item, 'REFUTED', 'REFUTED', {stage: 'GUARDIAN', step: 'validate_kernel_digest_attestation', reason: attestation.reason, nextOperation: 'REPAIR_KERNEL_DIGEST_EVIDENCE', blockedBy: ['kernel-before-digest', 'kernel-after-digest'], beforeDigest: computedBefore, afterDigest: computedAfter, artifactBeforeDigest: artifactBefore, artifactAfterDigest: artifactAfter}, {foundation_authorization_evaluated: false, foundation_receipt_evaluated: false});
}

async function runCase(item, guardian, guardianAuth, workflow, workspace, fixture, manifest, report) {
  if (item.mode === 'PASS_NULL_KERNEL_DIGEST' || item.mode === 'PASS_MISMATCH_KERNEL_DIGEST') return digestResult(guardian, item, fixture, item.mode);
  const fullWorkflowModes = new Set(['VALID_FOUNDATION_AUTHORIZATION', 'MISSING_FOUNDATION_AUTHORIZATION', 'CHANGED_PATH_TUPLE_MISMATCH', 'EXTRA_PROTECTED_PATH', 'STALE_FOUNDATION_AUTHORIZATION', 'MALFORMED_FOUNDATION_AUTHORIZATION', 'FOUNDATION_CARDINALITY_EXHAUSTED', 'FOUNDATION_REPLAY']);
  if (fullWorkflowModes.has(item.mode)) {
    const changedFiles = item.mode === 'MISSING_FOUNDATION_AUTHORIZATION' || item.mode === 'STALE_FOUNDATION_AUTHORIZATION' || item.mode === 'MALFORMED_FOUNDATION_AUTHORIZATION' || item.mode === 'FOUNDATION_CARDINALITY_EXHAUSTED' || item.mode === 'FOUNDATION_REPLAY' || item.mode === 'VALID_FOUNDATION_AUTHORIZATION' ? changedPathFiles(manifest, '') : changedPathFiles(manifest, item.mode);
    const pull = makePull(fixture, changedFiles);
    const outcome = await executeWorkflowScript(workflow, workspace, pull, fixture, manifest, changedFiles, item.mode, guardianAuth);
    return workflowResult(item, outcome, item.expected, true);
  }
  if (item.mode === 'UNPROTECTED_FEATURE') {
    const changedFiles = [{filename: 'docs/feature-pr.md', previous_filename: null, status: 'modified'}];
    const pull = makePull(fixture, changedFiles);
    const outcome = await executeWorkflowScript(workflow, workspace, pull, fixture, manifest, changedFiles, item.mode, guardianAuth);
    return workflowResult(item, outcome, item.expected, false);
  }
  throw new Error(`unsupported v3 Guardian harness mode ${item.mode}`);
}

function unsignedHarnessReport(report) {
  const copy = {...report};
  delete copy.artifact_digest;
  return JSON.stringify(copy);
}

async function main() {
  const values = parseArgs(process.argv.slice(2));
  for (const required of ['cases', 'report', 'out']) if (!values[required]) throw new Error(`--${required} is required`);
  const repoRoot = process.cwd();
  const cases = readJSON(values.cases);
  const report = readJSON(values.report);
  if (cases.migration_version !== 'v3' || !cases.fixture_v3) throw new Error('v3 Guardian cases artifact is required');
  const fixture = cases.fixture_v3;
  const {workspace, manifest} = await fetchFixture(repoRoot, fixture);
  const workflow = fs.readFileSync(path.join(workspace, '.github/workflows/ci-guardian.yml'), 'utf8');
  const guardian = require(path.join(workspace, 'scripts/ci-proof/guardian.js'));
  const guardianAuth = require(path.join(workspace, 'scripts/ci-proof/foundation_authorization.js'));
  const currentScript = extractFeatureScript(workflow);
  const results = [];
  for (const item of cases.cases) results.push(await runCase(item, guardian, guardianAuth, currentScript, workspace, fixture, manifest, report));
  const output = {
    schema: 'gooo/receipt-schema-migration/guardian-harness/v2',
    migration_version: 'v3',
    ir_digest: cases.ir_digest,
    fixture: {},
    fixture_v3: fixture,
    fixture_file_count: manifest.pinned_files.length,
    results,
    summary: {
      closed_count: results.filter((item) => item.state === 'CLOSED').length,
      unknown_count: results.filter((item) => item.state === 'UNKNOWN').length,
      refuted_count: results.filter((item) => item.state === 'REFUTED').length,
      foundation_authorization_count: results.filter((item) => item.evidence && item.evidence.foundation_authorization_evaluated).length,
      foundation_receipt_count: results.filter((item) => item.evidence && item.evidence.foundation_receipt_evaluated).length,
    },
  };
  output.artifact_digest = `sha256:${crypto.createHash('sha256').update(unsignedHarnessReport(output)).digest('hex')}`;
  writeJSON(values.out, output);
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exitCode = 1;
});
