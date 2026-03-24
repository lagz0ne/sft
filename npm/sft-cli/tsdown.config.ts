import { defineConfig } from "tsdown";

export default defineConfig({
  entry: {
    bin: "src/bin.ts",
    platform: "src/platform.ts",
  },
  format: ["esm"],
  platform: "node",
  target: "node16",
  clean: true,
  shims: true,
  minify: true,
});
