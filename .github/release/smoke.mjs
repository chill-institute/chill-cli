import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import semanticRelease from "semantic-release";

const analyzer = "@semantic-release/commit-analyzer";
const notesGenerator = "@semantic-release/release-notes-generator";
const preset = "conventionalcommits";
const releaseRules = [
  { breaking: true, release: "major" },
  { revert: true, release: "patch" },
  { type: "feat", release: "minor" },
  { type: "fix", release: "patch" },
  { type: "perf", release: "patch" },
  { type: "refactor", release: "patch" },
  { type: "revert", release: "patch" },
];
const presetConfig = {
  types: [
    { type: "feat", section: "Features" },
    { type: "fix", section: "Bug Fixes" },
    { type: "perf", section: "Performance Improvements" },
    { type: "refactor", section: "Code Refactoring" },
    { type: "revert", section: "Reverts" },
  ],
};

const config = JSON.parse(readFileSync(".releaserc.json", "utf8"));
const pluginName = (plugin) => (Array.isArray(plugin) ? plugin[0] : plugin);
const pluginOptions = (name) => {
  const plugin = config.plugins.find((entry) => pluginName(entry) === name);
  return Array.isArray(plugin) ? plugin[1] : undefined;
};
assert.deepEqual(
  pluginOptions(analyzer),
  { preset, releaseRules },
  `${analyzer} options drifted from the shared release policy`,
);
assert.deepEqual(
  pluginOptions(notesGenerator),
  { preset, presetConfig },
  `${notesGenerator} options drifted from the shared release policy`,
);
for (const plugin of config.plugins) {
  await import(pluginName(plugin));
}

const work = mkdtempSync(join(tmpdir(), "release-smoke-"));
try {
  const git = (...args) =>
    execFileSync("git", args, { cwd: work, stdio: "pipe" });
  const remote = join(work, "remote.git");
  execFileSync("git", [
    "init",
    "--quiet",
    "--bare",
    "--initial-branch=main",
    remote,
  ]);
  git("init", "--quiet", "--initial-branch=main");
  git("remote", "add", "origin", `file://${remote}`);
  const commit = (...messages) => {
    for (const message of messages) {
      git(
        "-c",
        "user.name=smoke",
        "-c",
        "user.email=smoke@example.invalid",
        "commit",
        "--quiet",
        "--allow-empty",
        "-m",
        message,
      );
    }
    git("push", "--quiet", "origin", "main");
  };

  const env = Object.fromEntries(
    Object.entries(process.env).filter(
      ([key]) => !["GITHUB_ACTIONS", "GITHUB_TOKEN", "GH_TOKEN"].includes(key),
    ),
  );
  const plan = async () => {
    const result = await semanticRelease(
      {
        ...config,
        plugins: config.plugins.filter((plugin) =>
          [analyzer, notesGenerator].includes(pluginName(plugin)),
        ),
        dryRun: true,
        ci: false,
      },
      { cwd: work, env },
    );
    return result ? result.nextRelease : undefined;
  };

  commit("chore: smoke base", "feat: smoke feature", "fix: smoke fix");
  const first = await plan();
  if (
    !first ||
    first.version !== "1.0.0" ||
    !first.notes.includes("smoke feature")
  ) {
    throw new Error(`Unexpected smoke release: ${JSON.stringify(first)}`);
  }
  git("-c", "tag.gpgSign=false", "tag", first.gitTag);
  git("push", "--quiet", "origin", first.gitTag);

  commit(
    "build: smoke build",
    "chore: smoke chore",
    "ci: smoke ci",
    "deps: smoke deps",
    "docs: smoke docs",
    "test: smoke test",
  );
  const quiet = await plan();
  if (quiet) {
    throw new Error(`Non-releasing types cut ${quiet.version}`);
  }

  commit("fix: smoke patch");
  const patch = await plan();
  if (!patch || patch.version !== "1.0.1") {
    throw new Error(`Unexpected smoke patch: ${JSON.stringify(patch)}`);
  }
  git("-c", "tag.gpgSign=false", "tag", patch.gitTag);
  git("push", "--quiet", "origin", patch.gitTag);

  commit("refactor: smoke refactor");
  const refactor = await plan();
  if (
    !refactor ||
    refactor.version !== "1.0.2" ||
    !refactor.notes.includes("smoke refactor")
  ) {
    throw new Error(`Unexpected smoke refactor: ${JSON.stringify(refactor)}`);
  }
  console.log(
    `release smoke ok: ${first.gitTag}, ${patch.gitTag}, ${refactor.gitTag}`,
  );
} finally {
  rmSync(work, { recursive: true, force: true });
}
