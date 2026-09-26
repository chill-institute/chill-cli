import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import semanticRelease from "semantic-release";

const config = JSON.parse(readFileSync(".releaserc.json", "utf8"));
const pluginName = (plugin) => (Array.isArray(plugin) ? plugin[0] : plugin);
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
  for (const message of [
    "chore: smoke base",
    "feat: smoke feature",
    "fix: smoke fix",
  ]) {
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

  const env = Object.fromEntries(
    Object.entries(process.env).filter(
      ([key]) => !["GITHUB_ACTIONS", "GITHUB_TOKEN", "GH_TOKEN"].includes(key),
    ),
  );
  const result = await semanticRelease(
    {
      ...config,
      plugins: config.plugins.filter(
        (plugin) => pluginName(plugin) !== "@semantic-release/github",
      ),
      dryRun: true,
      ci: false,
    },
    { cwd: work, env },
  );
  const next = result ? result.nextRelease : undefined;
  if (
    !next ||
    next.version !== "1.0.0" ||
    !next.notes.includes("smoke feature")
  ) {
    throw new Error(`Unexpected smoke release: ${JSON.stringify(next)}`);
  }
  console.log(`release smoke ok: ${next.gitTag}`);
} finally {
  rmSync(work, { recursive: true, force: true });
}
