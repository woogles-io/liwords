// Uploads the analyzer's files that wasm-files.json lists, from
// public/wasm/<dir>/, to the woogles.io bucket that serves them: only those
// the bucket lacks, or holds without S3's own SHA-256 checksum, each with its
// sha256 kept as the object's metadata (x-amz-meta-sha256), which
// fetch-wasm.mjs --check reads back before a build, and S3 checking that
// sha256 as it receives the file. Each local file is checked against the
// listed sha256 first. A file the bucket holds under the same name with
// another sha256 stops it before anything is sent: browsers and the CDN keep
// a served file for weeks, so a file's bytes never change under its name, and
// a changed file takes a new name or a new dir. Nothing is uploaded unless
// every file passes. It needs AWS credentials that may read and write the
// bucket, and it writes to production: a dry run, printing what it would
// send, unless --apply.
//   node scripts/upload-wasm.mjs           the dry run
//   node scripts/upload-wasm.mjs --apply   the upload
// --profile <name> uses that AWS profile, as the aws command's own --profile
// does; without it, the aws command picks one, as AWS_PROFILE says.
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ui = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const manifest = JSON.parse(
  await readFile(path.join(ui, "wasm-files.json"), "utf8"),
);
const dir = path.join(ui, "public", "wasm", manifest.dir);
const bucketName = "woogles.io";
const prefix = `wasm/${manifest.dir}/`;
const apply = process.argv.includes("--apply");
const at = process.argv.indexOf("--profile");
if (at >= 0 && !process.argv[at + 1]) {
  console.error("--profile needs the profile's name");
  process.exit(1);
}
const profile = at >= 0 ? ["--profile", process.argv[at + 1]] : [];

// what the bucket holds under a name, or null where it holds nothing
const held = (name) => {
  const run = spawnSync(
    "aws",
    [
      "s3api",
      "head-object",
      "--bucket",
      bucketName,
      "--key",
      `${prefix}${name}`,
      "--checksum-mode",
      "ENABLED",
      "--output",
      "json",
      ...profile,
    ],
    { encoding: "utf8" },
  );
  if (run.status !== 0) {
    if (/\(404\)/.test(run.stderr)) return null;
    console.error(`stopped: aws s3api head-object ${name}: ${run.stderr}`);
    process.exit(1);
  }
  const head = JSON.parse(run.stdout);
  return {
    sha256: head.Metadata?.sha256,
    checksum: Boolean(head.ChecksumSHA256),
  };
};

const commands = [];
const wrong = [];
let inPlace = 0;
for (const { name, size, sha256: sum } of manifest.files) {
  const file = path.join(dir, name);
  const bytes = await readFile(file).catch(() => null);
  if (!bytes) {
    wrong.push(`${file}: missing`);
    continue;
  }
  const got = createHash("sha256").update(bytes).digest("hex");
  if (bytes.length !== size || got !== sum) {
    wrong.push(`${file}: not the file wasm-files.json lists`);
    continue;
  }
  const there = held(name);
  if (there && there.sha256 !== sum) {
    wrong.push(`${prefix}${name}: the bucket holds other bytes under its name`);
    continue;
  }
  if (there && there.checksum) {
    inPlace++;
    continue;
  }
  // the cache the deploy gives the site's other files
  commands.push([
    "s3",
    "cp",
    file,
    `s3://${bucketName}/${prefix}${name}`,
    "--cache-control",
    "max-age=3024000",
    "--metadata",
    `sha256=${sum}`,
    // S3 refuses the file unless the bytes it receives have the sha256 the
    // aws command sends with them, taken from this file
    "--checksum-algorithm",
    "SHA256",
    ...profile,
  ]);
}
if (wrong.length) {
  console.error(`nothing uploaded:\n  ${wrong.join("\n  ")}`);
  process.exit(1);
}
for (const args of commands) {
  console.log(`aws ${args.join(" ")}`);
  if (!apply) continue;
  const run = spawnSync("aws", args, { stdio: "inherit" });
  if (run.status !== 0) {
    console.error(`stopped: aws exited ${run.status ?? run.error}`);
    process.exit(1);
  }
}
console.log(
  `${apply ? "uploaded" : "dry run:"} ${commands.length} files for s3://${bucketName}/${prefix}, ${inPlace} in place already${apply ? "" : "; --apply uploads them"}`,
);
