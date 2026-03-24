#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { getBinaryPath } from "./platform";

try {
  execFileSync(getBinaryPath(), process.argv.slice(2), { stdio: "inherit" });
} catch (e: unknown) {
  const err = e as { status?: number };
  if (err.status !== undefined) process.exit(err.status);
  throw e;
}
