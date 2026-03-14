import { copyFile, mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const siteRoot = path.resolve(__dirname, "..");
const repoRoot = path.resolve(siteRoot, "..");
const sourceDir = path.join(repoRoot, "images");
const targetDir = path.join(siteRoot, "public", "media");

const assets = [
  "prtr-banner.png",
  "prtr-setup-doctor.gif",
  "prtr-routing-history.gif",
  "prtr-delivery-paste.gif",
];

await mkdir(targetDir, { recursive: true });

await Promise.all(
  assets.map(async (name) => {
    await copyFile(path.join(sourceDir, name), path.join(targetDir, name));
  }),
);
