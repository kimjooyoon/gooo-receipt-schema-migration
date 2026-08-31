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

async function fetchFixture(repoRoot, fixture) {
  const manifestPath = path.join(repoRoot, fixture.manifest_path);
  const manifest = readJSON(manifestPath);
  if (manifest.repository !== fixture.repository || manifest.ref !== fixture.ref || manifest.commit !== fixture.commit || manifest.fetch_mode !== 'raw_at_immutable_commit_with_git_blob_sha1_verification') {
    throw new Error('Guardian fixture manifest is not bound to the generated fixture declaration');
  }
  const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'gooo-guardian-v2-'));
  for (const item of manifest.files) {
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
    if (line === '') {
      body.push('');
    } else if (line.startsWith('            ')) {
      body.push(line.slice(12));
    } else {
      break;
    }
  }
  const script = body.join('\n').replace(/\n+$/, '');
  if (!script.includes(marker) || !script.includes('let beforeDigest = null;') || !script.includes('computedBeforeDigest: beforeDigest')) {
    throw new Error('pinned feature script does not contain the expected Guardian scope probe');
  }
  return script;
}

function correctedScopeScript(current) {
  const inner = '\n  let beforeDigest = null;\n  let afterDigest = null;';
  if (!current.includes(inner)) throw new Error('current Guardian script did not expose the expected inner digest declaration');
  const moved = 'let beforeDigest = null;\nlet afterDigest = null;\n';
  return current.replace(inner, '').replace('const eventPull = context.payload.pull_request;', `${moved}const eventPull = context.payload.pull_request;`);
}

function makePull(fixtureCommit, changedPath) {
  return {
    number: 609,
    state: 'open',
    draft: false,
    changed_files: 1,
    base: {ref: 'dev', sha: fixtureCommit, repo: {full_name: 'kimjooyoon/meta-ontology-go'}},
    head: {ref: 'agent/v0.2-guardian-harness', sha: 'a'.repeat(40), repo: {full_name: 'kimjooyoon/meta-ontology-go'}},
    changed_path: changedPath,
  };
}

function mockGitHub(pull) {
  const repository = 'kimjooyoon/meta-ontology-go';
  const mainSHA = 'b'.repeat(40);
  return {
    rest: {
      git: {
        getRef: async ({ref}) => ({status: 200, data: {object: {sha: ref === 'heads/main' ? mainSHA : pull.base.sha}}}),
      },
      repos: {
        compareCommits: async () => ({status: 200, data: {status: 'ahead', ahead_by: 1, behind_by: 0, merge_base_commit: {sha: mainSHA}}}),
        getCommit: async () => ({status: 200, data: {commit: {tree: {sha: 'c'.repeat(40)}}}}),
        getTree: async () => ({status: 200, data: {sha: 'c'.repeat(40), truncated: false, tree: []}}),
        getContent: async () => ({status: 404, data: null}),
        getEnvironment: async () => ({status: 404, data: null}),
      },
      pulls: {
        get: async () => ({status: 200, data: pull}),
        listFiles: async () => ({status: 200, data: [{filename: pull.changed_path, status: 'modified'}]}),
      },
    },
  };
}

async function executeWorkflowScript(script, workspace, pull, fixtureCommit) {
  const scriptPath = path.join(workspace, 'workflow-script.js');
  fs.writeFileSync(scriptPath, script);
  const failures = [];
  const summary = {
    addHeading() { return this; },
    addRaw() { return this; },
    async write() { return undefined; },
  };
  const core = {
    summary,
    setFailed(message) { failures.push(String(message)); },
    setOutput() {},
  };
  const context = {
    payload: {
      action: 'synchronize',
      pull_request: pull,
      repository: {full_name: 'kimjooyoon/meta-ontology-go', default_branch: 'dev'},
    },
    repo: {owner: 'kimjooyoon', repo: 'meta-ontology-go'},
  };
  const workflowSHA = fixtureCommit;
  const sandbox = {
    Buffer,
    console,
    context,
    core,
    github: mockGitHub(pull),
    process: {env: {
      GUARDIAN_WORKFLOW_REF: 'kimjooyoon/meta-ontology-go/.github/workflows/ci-guardian.yml@refs/heads/dev',
      GUARDIAN_WORKFLOW_SHA: workflowSHA,
      GUARDIAN_RUNTIME_REF: 'refs/heads/dev',
      GUARDIAN_RUNTIME_SHA: workflowSHA,
      GUARDIAN_RUN_ID: '3405',
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
  try {
    return readJSON(file);
  } catch {
    return null;
  }
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

function digestResult(guardian, item, mode) {
  const before = `sha256:${'a'.repeat(64)}`;
  const after = `sha256:${'b'.repeat(64)}`;
  let computedBefore = before;
  let computedAfter = after;
  let artifactBefore = before;
  let artifactAfter = after;
  if (mode === 'NULL_DIGEST') {
    computedBefore = null;
    computedAfter = null;
    artifactBefore = null;
    artifactAfter = null;
  }
  if (mode === 'DIGEST_MISMATCH') artifactAfter = `sha256:${'c'.repeat(64)}`;
  const attestation = guardian.validateKernelDigestAttestation({
    kernelPaths: ['scripts/ci-proof/guardian.js'],
    computedBeforeDigest: computedBefore,
    computedAfterDigest: computedAfter,
    artifactBeforeDigest: artifactBefore,
    artifactAfterDigest: artifactAfter,
  });
  if (mode === 'DIGEST_MATCH') {
    if (attestation.decision !== 'PASS') throw new Error(`matching digest case did not pass: ${attestation.reason}`);
    return resultFor(item, 'CLOSED', 'PASS', {stage: 'GUARDIAN', step: 'validate_kernel_digest_attestation', reason: attestation.reason, nextOperation: 'NONE', blockedBy: [], beforeDigest: computedBefore, afterDigest: computedAfter, artifactBeforeDigest: artifactBefore, artifactAfterDigest: artifactAfter});
  }
  if (attestation.decision !== 'REFUTED') throw new Error(`digest case did not refute: ${mode}`);
  return resultFor(item, 'REFUTED', 'REFUTED', {stage: 'GUARDIAN', step: 'validate_kernel_digest_attestation', reason: attestation.reason, nextOperation: 'REPAIR_KERNEL_DIGEST_EVIDENCE', blockedBy: ['kernel-before-digest', 'kernel-after-digest'], beforeDigest: computedBefore, afterDigest: computedAfter, artifactBeforeDigest: artifactBefore, artifactAfterDigest: artifactAfter});
}

async function runCase(item, guardian, workflow, corrected, workspace, fixtureCommit, report) {
  switch (item.mode) {
    case 'CURRENT_SNAPSHOT': {
      const outcome = await executeWorkflowScript(workflow, workspace, makePull(fixtureCommit, 'docs/feature-pr.md'), fixtureCommit);
      if (!outcome.error || outcome.error.name !== 'ReferenceError' || outcome.error.message !== 'beforeDigest is not defined') throw new Error('current snapshot did not reproduce ReferenceError: beforeDigest is not defined');
      const reason = `${outcome.error.name}: ${outcome.error.message}`;
      return resultFor(item, 'REFUTED', 'REFERENCE_ERROR', {stage: 'GUARDIAN', step: 'execute_actual_pull_request_target_script', reason, nextOperation: 'INITIALIZE_DIGEST_VARIABLES_BEFORE_POLICY_BRANCH', blockedBy: ['workflow-scope']}, {workflow_source: 'base-controlled github.workflow_sha', candidate_source: 'candidate-controlled pull_request.files'});
    }
    case 'CORRECTED_SCOPE_PASS_ARTIFACT_NON_NULL':
    case 'SEPARATE_SOURCES': {
      const outcome = await executeWorkflowScript(corrected, workspace, makePull(fixtureCommit, 'docs/feature-pr.md'), fixtureCommit);
      if (outcome.error || !outcome.artifact || outcome.artifact.decision !== 'PASS' || outcome.failures.length !== 0) throw new Error(`corrected feature PR did not close: ${outcome.error ? outcome.error.message : outcome.failures.join(';')}`);
      return resultFor(item, 'CLOSED', 'PASS', {stage: 'GUARDIAN', step: 'validate_pass_artifact', reason: 'PASS artifact is non-null after corrected digest variable scope', nextOperation: 'NONE', blockedBy: []}, {artifact_non_null: true, workflow_source: 'base-controlled github.workflow_sha', candidate_source: 'candidate-controlled pull_request.files', scope: 'beforeDigest/afterDigest initialized before policy branch'});
    }
    case 'NULL_DIGEST':
    case 'DIGEST_MISMATCH':
    case 'DIGEST_MATCH':
      return digestResult(guardian, item, item.mode);
    case 'PROTECTED_MIGRATION_PATH': {
      const pull = makePull(fixtureCommit, 'internal/verify/scope_schema_coherence_migration_adoption_20260831.go');
      const observed = await guardian.inspectChangedFiles({
        owner: 'kimjooyoon',
        repo: 'meta-ontology-go',
        baseRepoFullName: 'kimjooyoon/meta-ontology-go',
        pullNumber: pull.number,
        expectedCount: 1,
        listFiles: async () => ({status: 200, data: [{filename: pull.changed_path, status: 'modified'}]}),
      });
      if (observed.decision !== 'FAIL_CLOSED' || observed.kernelPaths.length !== 1) throw new Error('protected migration path did not fail closed');
      return resultFor(item, 'REFUTED', 'REFUTED', {stage: 'GUARDIAN', step: 'inspect_candidate_changed_paths', reason: observed.reason, nextOperation: 'REQUIRE_AUTHORIZED_PROTECTED_PATH_FLOW', blockedBy: ['protected-kernel-path']}, {candidate_path: pull.changed_path, base_controlled_workflow: true});
    }
    case 'UNSUPPORTED_FUTURE_SCHEMA': {
      const scenario = report.scenarios.find((entry) => entry.scenario_id === 'UNKNOWN_UNSUPPORTED_FUTURE_SCHEMA');
      if (!scenario || scenario.state !== 'UNKNOWN' || !scenario.claim || Object.keys(scenario.claim).sort().join(',') !== 'blocked_by,next_operation,reason,stage,state,step,unknown_class') throw new Error('generated future-schema scenario did not preserve the exact UNKNOWN six-field claim');
      const c = scenario.claim;
      return resultFor(item, 'UNKNOWN', 'UNKNOWN', {stage: c.stage, step: c.step, reason: c.reason, unknownClass: c.unknown_class, nextOperation: c.next_operation, blockedBy: c.blocked_by}, {generated_scenario_state: scenario.state, generated_scenario_id: scenario.scenario_id});
    }
    default:
      throw new Error(`unsupported Guardian harness mode ${item.mode}`);
  }
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
  const fixture = cases.fixture;
  const {workspace, manifest} = await fetchFixture(repoRoot, fixture);
  const workflow = fs.readFileSync(path.join(workspace, '.github/workflows/ci-guardian.yml'), 'utf8');
  const currentScript = extractFeatureScript(workflow);
  const corrected = correctedScopeScript(currentScript);
  const guardian = require(path.join(workspace, 'scripts/ci-proof/guardian.js'));
  const results = [];
  for (const item of cases.cases) results.push(await runCase(item, guardian, currentScript, corrected, workspace, fixture.commit, report));
  const output = {
    schema: 'gooo/receipt-schema-migration/guardian-harness/v1',
    migration_version: 'v2',
    ir_digest: cases.ir_digest,
    fixture,
    fixture_file_count: manifest.files.length,
    results,
    summary: {
      closed_count: results.filter((item) => item.state === 'CLOSED').length,
      unknown_count: results.filter((item) => item.state === 'UNKNOWN').length,
      refuted_count: results.filter((item) => item.state === 'REFUTED').length,
    },
  };
  output.artifact_digest = `sha256:${crypto.createHash('sha256').update(unsignedHarnessReport(output)).digest('hex')}`;
  writeJSON(values.out, output);
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exitCode = 1;
});
