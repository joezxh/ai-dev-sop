// Verification script for the installed ECC .cursor/ configuration.
const fs = require('fs');
const path = require('path');

const ROOT = 'd:/projects/ai-dev-sop/.cursor';
const PASS = [];
const FAIL = [];

function ok(msg) { PASS.push(msg); console.log('  [PASS] ' + msg); }
function fail(msg) { FAIL.push(msg); console.log('  [FAIL] ' + msg); }

console.log('=== Verifying ECC install in d:/projects/ai-dev-sop/.cursor/ ===\n');

// Rules
console.log('Rules:');
const rulesDir = path.join(ROOT, 'rules');
if (!fs.existsSync(rulesDir)) {
  fail('rules directory missing');
} else {
  const ruleFiles = fs.readdirSync(rulesDir).filter(f => f.endsWith('.mdc'));
  if (ruleFiles.length >= 39) ok(`rules/ has ${ruleFiles.length} .mdc files (expected ≥ 39)`);
  else fail(`rules/ has only ${ruleFiles.length} .mdc files`);
  // validate frontmatter anywhere in the first 6 lines
  let fmOk = 0;
  const fmFailures = [];
  for (const f of ruleFiles) {
    const head = fs.readFileSync(path.join(rulesDir, f), 'utf8').split('\n').slice(0, 6).join('\n');
    if (/^---\s*$/.test(head.split('\n')[0]) && /^(description|name):/m.test(head) && /^---\s*$/m.test(head.split('\n').slice(1).join('\n'))) {
      fmOk++;
    } else {
      fmFailures.push(f);
    }
  }
  if (fmOk === ruleFiles.length) ok(`All ${ruleFiles.length} rules have valid YAML frontmatter`);
  else fail(`${ruleFiles.length - fmOk}/${ruleFiles.length} rules missing frontmatter. Problematic: ${fmFailures.join(', ')}`);
}

// Hooks scripts
console.log('\nHook scripts:');
const hooksDir = path.join(ROOT, 'hooks');
if (!fs.existsSync(hooksDir)) {
  fail('hooks directory missing');
} else {
  const hookFiles = fs.readdirSync(hooksDir).filter(f => f.endsWith('.js'));
  if (hookFiles.length >= 17) ok(`hooks/ has ${hookFiles.length} .js files (expected 17)`);
  else fail(`hooks/ has only ${hookFiles.length} .js files`);
  // proper syntax check via Node's vm module (parses only, does not execute)
  const vm = require('vm');
  let syntaxOk = 0;
  const syntaxFailures = [];
  for (const f of hookFiles) {
    const src = fs.readFileSync(path.join(hooksDir, f), 'utf8');
    try {
      new vm.Script(src, { filename: f });
      syntaxOk++;
    } catch (e) {
      syntaxFailures.push(`${f}: ${e.message.split('\n')[0]}`);
    }
  }
  if (syntaxOk === hookFiles.length) ok(`All ${hookFiles.length} hook scripts pass Node syntax validation`);
  else fail(`${hookFiles.length - syntaxOk}/${hookFiles.length} hook scripts have syntax errors. ${syntaxFailures.join('; ')}`);
}

// hooks.json
console.log('\nhooks.json:');
const hooksJsonPath = path.join(ROOT, 'hooks.json');
if (fs.existsSync(hooksJsonPath)) {
  try {
    const h = JSON.parse(fs.readFileSync(hooksJsonPath, 'utf8'));
    if (h.version === 1 && h.hooks && typeof h.hooks === 'object') {
      const eventCount = Object.keys(h.hooks).length;
      const cmdCount = Object.values(h.hooks).reduce((s, arr) => s + arr.length, 0);
      ok(`hooks.json is valid (version=${h.version}, ${eventCount} events, ${cmdCount} commands)`);
    } else fail('hooks.json does not have expected shape {version:1, hooks:{...}}');
  } catch (e) { fail(`hooks.json invalid JSON: ${e.message}`); }
} else fail('hooks.json missing');

// Skills
console.log('\nSkills:');
const skillsDir = path.join(ROOT, 'skills');
if (!fs.existsSync(skillsDir)) {
  fail('skills directory missing');
} else {
  const skillDirs = fs.readdirSync(skillsDir, { withFileTypes: true }).filter(e => e.isDirectory());
  if (skillDirs.length >= 10) ok(`skills/ has ${skillDirs.length} skill subdirectories (expected 10)`);
  else fail(`skills/ has only ${skillDirs.length} skill subdirectories`);
  // each skill must have SKILL.md
  let allHaveSkill = true;
  for (const s of skillDirs) {
    const skillMd = path.join(skillsDir, s.name, 'SKILL.md');
    if (!fs.existsSync(skillMd)) { console.log(`         missing: ${s.name}/SKILL.md`); allHaveSkill = false; }
  }
  if (allHaveSkill) ok(`All ${skillDirs.length} skills have SKILL.md`);
  else fail('some skills missing SKILL.md');
}

// mcp.json
console.log('\nmcp.json:');
const mcpPath = path.join(ROOT, 'mcp.json');
if (fs.existsSync(mcpPath)) {
  try {
    const m = JSON.parse(fs.readFileSync(mcpPath, 'utf8'));
    if (m.mcpServers && m.mcpServers['chrome-devtools']) ok('mcp.json valid; chrome-devtools MCP server registered');
    else fail('mcp.json missing chrome-devtools server');
  } catch (e) { fail(`mcp.json invalid JSON: ${e.message}`); }
} else fail('mcp.json missing');

console.log(`\n=== Summary: ${PASS.length} PASS, ${FAIL.length} FAIL ===`);
if (FAIL.length === 0) console.log('All ECC capabilities are installed and validated.');
else { console.log('\nFAILURES:'); FAIL.forEach(f => console.log('  - ' + f)); process.exit(1); }
