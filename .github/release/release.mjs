import { appendFileSync } from "node:fs";
import semanticRelease from "semantic-release";

const args = new Set(process.argv.slice(2));
const unknown = [...args].filter(
  (arg) => arg !== "--dry-run" && arg !== "--no-ci",
);
if (unknown.length > 0) {
  throw new Error(`Unknown arguments: ${unknown.join(" ")}`);
}

const options = {};
if (args.has("--dry-run")) options.dryRun = true;
if (args.has("--no-ci")) options.ci = false;

const result = await semanticRelease(options);
const next = result ? result.nextRelease : undefined;
const outputs = next
  ? {
      new_release_published: "true",
      new_release_version: next.version,
      new_release_git_tag: next.gitTag,
    }
  : { new_release_published: "false" };

const lines = Object.entries(outputs).map(([key, value]) => `${key}=${value}`);
console.log(lines.join("\n"));
if (process.env.GITHUB_OUTPUT) {
  appendFileSync(process.env.GITHUB_OUTPUT, `${lines.join("\n")}\n`);
}
