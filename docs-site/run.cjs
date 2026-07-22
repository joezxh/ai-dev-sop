#!/usr/bin/env node
/**
 * run — convenience launcher for docs-site scripts.
 *
 * Usage:
 *   node run.cjs          → show this help
 *   node run.cjs dev      → pnpm dev     (VitePress dev server)
 *   node run.cjs build    → pnpm build   (production build)
 *   node run.cjs preview  → pnpm preview  (preview build)
 *   node run.cjs lint     → pnpm lint:links
 *   node run.cjs sync     → pnpm sync-docs
 *   node run.cjs <any>    → pass through to pnpm
 */
const { spawn, execSync } = require("child_process");
const path = require("path");

const ALIASES = {
  dev:     "dev",
  build:   "build",
  preview: "preview",
  lint:    "lint:links",
  sync:    "sync-docs",
};

const HELP = `docs-site run launcher

Usage: node run.cjs <command>

Commands:
  dev     VitePress dev server
  build   Production build
  preview Preview production build
  lint    Check dead links
  sync    Sync docs from source

Examples:
  node run.cjs dev
  node run.cjs build

Or use pnpm directly:
  pnpm dev
  pnpm run dev
`;

function hasPnpm() {
  try {
    execSync("pnpm --version", { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function main() {
  const cmd = process.argv[2];

  if (!cmd || cmd === "--help" || cmd === "-h") {
    console.log(HELP);
    process.exit(0);
  }

  if (!hasPnpm()) {
    console.error(
      "Error: 'pnpm' not found. Install it with: npm install -g pnpm\n" +
      "Or use pnpm directly: pnpm " + cmd
    );
    process.exit(1);
  }

  const script = ALIASES[cmd] ?? cmd;
  const extraArgs = process.argv.slice(3);

  const child = spawn("pnpm", [script, ...extraArgs], {
    stdio: "inherit",
    shell: true,
    cwd: path.resolve(__dirname),
  });

  child.on("exit", (code) => process.exit(code ?? 0));
}

main();
