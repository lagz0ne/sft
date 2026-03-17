#!/usr/bin/env node
const { execFileSync } = require("child_process");
const { getBinaryPath } = require("./platform");

try {
  execFileSync(getBinaryPath(), process.argv.slice(2), { stdio: "inherit" });
} catch (e) {
  if (e.status !== undefined) process.exit(e.status);
  throw e;
}
