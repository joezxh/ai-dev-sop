// Replicate ECC cursor-project.js logic to install .cursor/* into the project root.
const fs = require('fs');
const path = require('path');

const ECC_ROOT = 'D:/projects/github/everything-claude-code';
const PROJECT_ROOT = 'd:/projects/ai-dev-sop';
const TARGET_ROOT = path.join(PROJECT_ROOT, '.cursor');

const ECC_CURSOR = path.join(ECC_ROOT, '.cursor');

function toCursorRuleFileName(fileName, sourceRelativeFile) {
  if (path.basename(sourceRelativeFile).toLowerCase() === 'readme.md') return null;
  return fileName.endsWith('.md') ? `${fileName.slice(0, -3)}.mdc` : fileName;
}

function copyFile(src, dest) {
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.copyFileSync(src, dest);
  return dest;
}

function copyFlatDir({ srcDir, destDir, nameTransform }) {
  if (!fs.existsSync(srcDir)) return [];
  const ops = [];
  const seen = new Set();
  for (const entry of fs.readdirSync(srcDir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    if (!entry.isFile()) continue;
    const newName = nameTransform(entry.name, entry.name);
    if (!newName) continue;
    if (seen.has(newName)) continue;
    seen.add(newName);
    ops.push(copyFile(path.join(srcDir, entry.name), path.join(destDir, newName)));
  }
  return ops;
}

function copySkills({ srcDir, destDir }) {
  if (!fs.existsSync(srcDir)) return [];
  const ops = [];
  for (const entry of fs.readdirSync(srcDir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    if (!entry.isDirectory()) continue;
    const skillSrc = path.join(srcDir, entry.name);
    const skillDest = path.join(destDir, entry.name);
    fs.mkdirSync(skillDest, { recursive: true });
    for (const file of fs.readdirSync(skillSrc, { withFileTypes: true })) {
      if (file.isFile()) {
        copyFile(path.join(skillSrc, file.name), path.join(skillDest, file.name));
      }
    }
    ops.push(skillDest);
  }
  return ops;
}

console.log('=== Installing ECC into .cursor/ ===\n');

// 1. Rules: src .cursor/rules/*.md → dst .cursor/rules/*.mdc
console.log('[1/4] Installing rules (.md → .mdc)...');
const rulesSrc = path.join(ECC_CURSOR, 'rules');
const rulesDest = path.join(TARGET_ROOT, 'rules');
const rulesInstalled = copyFlatDir({ srcDir: rulesSrc, destDir: rulesDest, nameTransform: toCursorRuleFileName });
console.log(`       Installed ${rulesInstalled.length} rule files:`);
rulesInstalled.forEach(p => console.log(`         - ${path.relative(PROJECT_ROOT, p)}`));

// 2. Hooks
console.log('\n[2/4] Installing hooks scripts...');
const hooksSrc = path.join(ECC_CURSOR, 'hooks');
const hooksDest = path.join(TARGET_ROOT, 'hooks');
let hooksInstalled = [];
if (fs.existsSync(hooksSrc)) {
  for (const entry of fs.readdirSync(hooksSrc, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    if (entry.isFile()) {
      hooksInstalled.push(copyFile(path.join(hooksSrc, entry.name), path.join(hooksDest, entry.name)));
    }
  }
}
console.log(`       Installed ${hooksInstalled.length} hook scripts:`);
hooksInstalled.forEach(p => console.log(`         - ${path.relative(PROJECT_ROOT, p)}`));

// 3. Skills
console.log('\n[3/4] Installing Cursor-targeted skills...');
const skillsSrc = path.join(ECC_CURSOR, 'skills');
const skillsDest = path.join(TARGET_ROOT, 'skills');
const skillsInstalled = copySkills({ srcDir: skillsSrc, destDir: skillsDest });
console.log(`       Installed ${skillsInstalled.length} skill directories:`);
skillsInstalled.forEach(p => console.log(`         - ${path.relative(PROJECT_ROOT, p)}`));

console.log('\n=== Done ===');
