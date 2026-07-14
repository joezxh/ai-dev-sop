// Sync docs/cn/*.md (and selected subdir docs) into docs-site/sop & docs-site/guide
// so VitePress's default srcDir (= repo root of the VitePress project) can serve them.
// Run via:  pnpm run sync-docs

import { copyFileSync, mkdirSync, existsSync, readdirSync, statSync } from 'node:fs'
import { dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const ROOT = dirname(dirname(dirname(__filename)))             // D:/projects/ai-dev-sop (scripts is under docs-site/scripts)
const SRC = join(ROOT, 'docs', 'cn')                            // docs/cn
const DST_SOP = join(ROOT, 'docs-site', 'sop')                  // docs-site/sop
const DST_GUIDE = join(ROOT, 'docs-site', 'guide')              // docs-site/guide
const DST_REF = join(ROOT, 'docs-site', 'reference')            // docs-site/reference
const DST_SITE = join(ROOT, 'docs-site')                        // docs-site root

function ensureDir(dir) {
  if (!existsSync(dir)) mkdirSync(dir, { recursive: true })
}

function listMarkdown(dir) {
  if (!existsSync(dir)) return []
  return readdirSync(dir).filter((f) => f.endsWith('.md'))
}

function copyIfNewer(src, dst) {
  if (!existsSync(dst)) {
    copyFileSync(src, dst)
    return 'created'
  }
  const a = statSync(src).mtimeMs
  const b = statSync(dst).mtimeMs
  if (a > b) {
    copyFileSync(src, dst)
    return 'updated'
  }
  return 'skipped'
}

function syncFile(rel, targetRoot) {
  const srcPath = join(SRC, rel)
  const dstPath = join(targetRoot, rel)
  if (!existsSync(srcPath)) return
  ensureDir(dirname(dstPath))
  const status = copyIfNewer(srcPath, dstPath)
  console.log(`[${status}] ${rel} -> ${relative(ROOT, dstPath)}`)
}

/** Copy a source file (relative to docs/cn/) to an arbitrary destination path (relative to docs-site/). */
function copyToDest(srcRel, destRel) {
  const srcPath = join(SRC, srcRel)
  const dstPath = join(DST_SITE, destRel)
  if (!existsSync(srcPath)) return
  ensureDir(dirname(dstPath))
  const status = copyIfNewer(srcPath, dstPath)
  console.log(`[${status}] alias ${srcRel} -> ${destRel}`)
}

function main() {
  ensureDir(DST_SOP)
  ensureDir(DST_GUIDE)

  // Core SOP source — develop-sop.md becomes /sop/ (the default SOP landing page)
  syncFile('develop-sop.md', DST_SOP)

  // Pipeline / scene / template docs live under sop/
  for (const f of listMarkdown(SRC)) {
    if (f === 'develop-sop.md') continue
    // Heuristic: anything that smells like a pipeline / scene / template goes to /sop/
    if (
      f.endsWith('-pipeline.md') ||
      f === 'scene-template.md' ||
      f.endsWith('-scene.md') ||
      f === 'qa-to-dev-complete-solution.md' ||
      f === 'AI开发skill字典.md' ||
      f === 'repo-wiki-tech.md'
    ) {
      syncFile(f, DST_SOP)
    } else {
      // Everything else (sketch / scratch) -> guide/
      syncFile(f, DST_GUIDE)
    }
  }

  // benchmark/ subfolder -> sop/benchmark/
  const bench = join(SRC, 'benchmark')
  if (existsSync(bench)) {
    for (const f of readdirSync(bench)) {
      const rel = join('benchmark', f)
      const srcFull = join(bench, f)
      if (statSync(srcFull).isFile() && rel.endsWith('.md')) {
        syncFile(rel, DST_SOP)
      }
    }
  }

  // --- Alias copies: make files available at their sidebar / nav URL paths ---
  // develop-sop.md → /sop/, /guide/, /reference/
  copyToDest('develop-sop.md', 'sop/index.md')
  copyToDest('develop-sop.md', 'guide/index.md')
  copyToDest('develop-sop.md', 'reference/index.md')
  // qa-to-dev-complete-solution.md → /guide/intro
  copyToDest('qa-to-dev-complete-solution.md', 'guide/intro.md')
  // AI开发skill字典.md → /guide/skills
  copyToDest('AI开发skill字典.md', 'guide/skills.md')
  // reference/ pipeline & template aliases
  copyToDest('scene-template.md', 'reference/scene-template.md')
  copyToDest('one-sentence-pipeline.md', 'reference/one-sentence-pipeline.md')
  copyToDest('framework-pipeline.md', 'reference/framework-pipeline.md')
  copyToDest('docs-pipeline.md', 'reference/docs-pipeline.md')
  copyToDest('copy-web-pipeline.md', 'reference/copy-web-pipeline.md')
  copyToDest('copy-app-pipeline.md', 'reference/copy-app-pipeline.md')
  copyToDest('java-upgrade-pipeline.md', 'reference/java-upgrade-pipeline.md')
}

main()
