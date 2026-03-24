import { createRequire } from "node:module";
import os from "node:os";
import path from "node:path";

const require = createRequire(import.meta.url);

const PLATFORMS: Record<string, string> = {
  "darwin arm64": "sft-cli-darwin-arm64",
  "darwin x64": "sft-cli-darwin-x64",
  "linux arm64": "sft-cli-linux-arm64",
  "linux x64": "sft-cli-linux-x64",
  "win32 x64": "@sft-cli/win32-x64",
};

export function getBinaryPath(): string {
  if (process.env.SFT_BINARY_PATH) return process.env.SFT_BINARY_PATH;

  const key = `${process.platform} ${os.arch()}`;
  const pkg = PLATFORMS[key];
  if (!pkg) {
    throw new Error(
      `Unsupported platform: ${key}. Supported: ${Object.keys(PLATFORMS).join(", ")}`,
    );
  }

  const bin = process.platform === "win32" ? "sft.exe" : "sft";
  try {
    return path.join(
      path.dirname(require.resolve(`${pkg}/package.json`)),
      "bin",
      bin,
    );
  } catch {
    throw new Error(
      `The package "${pkg}" could not be found. Make sure you don't use --no-optional when installing.`,
    );
  }
}
