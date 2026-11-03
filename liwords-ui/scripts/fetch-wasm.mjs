// The analyzer's word graphs and leave tables that git does not keep:
// wasm-files.json lists them, each with its size and sha256, and this fetches
// them from where woogles.io serves them into public/wasm/<dir>/, where the
// page, and the servers in docker-compose.yml (through data/lexica/gaddag),
// read them. A file already there and whole is left alone, so a run with
// every file in place fetches nothing.
//   node scripts/fetch-wasm.mjs           as npm start runs it: a file it
//                                         cannot fetch is named, and the
//                                         start goes on
//   node scripts/fetch-wasm.mjs --strict  as npm run fetch-wasm and CI run
//                                         it: exits 1 if any file is missing
//   node scripts/fetch-wasm.mjs --write   lists the files public/wasm/<dir>/
//                                         holds in wasm-files.json again
// WOOGLES_WASM_BASE, when set, is fetched from in place of the manifest's
// base, which ends with a slash, as it must.
import { createHash } from "node:crypto";
import { mkdir, readFile, readdir, rename, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ui = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const manifestPath = path.join(ui, "wasm-files.json");
const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
const dir = path.join(ui, "public", "wasm", manifest.dir);
const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");

if (process.argv.includes("--write")) {
  // every file but the ones git keeps there (.gitkeep) and a fetch cut short
  const names = (await readdir(dir))
    .filter((name) => !name.startsWith(".") && !name.endsWith(".part"))
    .sort();
  const files = [];
  for (const name of names) {
    const bytes = await readFile(path.join(dir, name));
    files.push({ name, size: bytes.length, sha256: sha256(bytes) });
  }
  await writeFile(
    manifestPath,
    `${JSON.stringify({ ...manifest, files }, null, 2)}\n`,
  );
  console.log(`wasm-files.json lists ${files.length} files`);
} else {
  const strict = process.argv.includes("--strict");
  const base = process.env.WOOGLES_WASM_BASE || manifest.base;
  await mkdir(dir, { recursive: true });
  const missing = [];
  let fetched = 0;
  for (const { name, size, sha256: sum } of manifest.files) {
    const file = path.join(dir, name);
    const have = await readFile(file).catch(() => null);
    if (have && have.length === size && sha256(have) === sum) continue;
    const url = `${base}${manifest.dir}/${name}`;
    try {
      const response = await fetch(url);
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const bytes = Buffer.from(await response.arrayBuffer());
      if (bytes.length !== size || sha256(bytes) !== sum) {
        throw new Error("not the file wasm-files.json lists");
      }
      // written whole, or not at all
      await writeFile(`${file}.part`, bytes);
      await rename(`${file}.part`, file);
      fetched++;
    } catch (e) {
      missing.push(`${url}: ${e instanceof Error ? e.message : e}`);
    }
  }
  if (fetched) {
    console.log(`fetched ${fetched} of public/wasm/${manifest.dir}'s files`);
  }
  if (missing.length) {
    console.error(
      `public/wasm/${manifest.dir} lacks ${missing.length} of its ${manifest.files.length} files:\n  ${missing.join("\n  ")}`,
    );
    if (strict) process.exit(1);
  }
}
