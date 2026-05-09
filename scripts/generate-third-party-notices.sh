#!/usr/bin/env bash
#
# Generate a release-ready third-party dependency notice summary.
#
# The output is intentionally concise: it records dependency names, versions,
# detected license identifiers, and source/homepage hints that can travel with
# binary and container releases.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${ROOT_DIR}/THIRD_PARTY_NOTICES.md"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

cd "${ROOT_DIR}/core"
go list -m -json all > "${TMP_DIR}/go-modules.jsonstream"

cd "${ROOT_DIR}/web"
pnpm licenses list --prod --json > "${TMP_DIR}/web-licenses.json"

ROOT_DIR="${ROOT_DIR}" OUTPUT="${OUTPUT}" TMP_DIR="${TMP_DIR}" node <<'NODE'
const fs = require("fs");
const path = require("path");

const rootDir = process.env.ROOT_DIR;
const output = process.env.OUTPUT;
const tmpDir = process.env.TMP_DIR;

function parseJSONStream(text) {
  const values = [];
  let depth = 0;
  let start = -1;
  let inString = false;
  let escaped = false;

  for (let i = 0; i < text.length; i++) {
    const ch = text[i];

    if (inString) {
      if (escaped) {
        escaped = false;
      } else if (ch === "\\") {
        escaped = true;
      } else if (ch === "\"") {
        inString = false;
      }
      continue;
    }

    if (ch === "\"") {
      inString = true;
      continue;
    }

    if (ch === "{") {
      if (depth === 0) start = i;
      depth++;
    } else if (ch === "}") {
      depth--;
      if (depth === 0 && start >= 0) {
        values.push(JSON.parse(text.slice(start, i + 1)));
        start = -1;
      }
    }
  }

  return values;
}

function firstExistingFile(dir, names, stopDir) {
  if (!dir) return null;

  let current = dir;
  while (current && current.startsWith(stopDir)) {
    for (const name of names) {
      const candidate = path.join(current, name);
      if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) {
        return candidate;
      }
    }

    const parent = path.dirname(current);
    if (parent === current) {
      break;
    }
    current = parent;
  }

  return null;
}

function detectLicenseFromText(text) {
  const normalized = text.toLowerCase();

  if (normalized.includes("apache license") && normalized.includes("version 2.0")) {
    return "Apache-2.0";
  }
  if (normalized.includes("mit license") || normalized.includes("permission is hereby granted, free of charge")) {
    return "MIT";
  }
  if (normalized.includes("bsd 3-clause") || normalized.includes("neither the name") && normalized.includes("contributors may be used to endorse")) {
    return "BSD-3-Clause";
  }
  if (normalized.includes("bsd 2-clause") || normalized.includes("redistribution and use in source and binary forms")) {
    return "BSD";
  }
  if (normalized.includes("mozilla public license") && normalized.includes("2.0")) {
    return "MPL-2.0";
  }
  if (normalized.includes("gnu lesser general public license")) {
    return "LGPL";
  }
  if (normalized.includes("gnu affero general public license")) {
    return "AGPL";
  }
  if (normalized.includes("gnu general public license")) {
    return "GPL";
  }
  if (normalized.includes("isc license") || normalized.includes("the isc license")) {
    return "ISC";
  }

  return "UNKNOWN";
}

function detectGoLicense(mod) {
  if (!mod.Dir) return "UNKNOWN";

  const modCache = path.join(process.env.HOME || "", "go", "pkg", "mod");
  const licenseFile = firstExistingFile(mod.Dir, [
    "LICENSE",
    "LICENSE.md",
    "LICENSE.txt",
    "COPYING",
    "COPYING.md",
    "COPYING.txt",
    "NOTICE",
  ], modCache);

  if (!licenseFile) return "UNKNOWN";

  const text = fs.readFileSync(licenseFile, "utf8").slice(0, 24000);
  return detectLicenseFromText(text);
}

function moduleSource(pathName) {
  if (pathName.startsWith("github.com/")) return `https://${pathName}`;
  if (pathName.startsWith("golang.org/x/")) return `https://${pathName}`;
  if (pathName.startsWith("google.golang.org/")) return `https://${pathName}`;
  if (pathName.startsWith("go.opentelemetry.io/")) return `https://${pathName}`;
  if (pathName.startsWith("go.uber.org/")) return `https://${pathName}`;
  return "";
}

function escapeCell(value) {
  return String(value || "").replace(/\|/g, "\\|").replace(/\n/g, " ");
}

function table(rows, columns) {
  const header = `| ${columns.map((c) => c.label).join(" | ")} |`;
  const sep = `| ${columns.map(() => "---").join(" | ")} |`;
  const body = rows.map((row) => {
    return `| ${columns.map((c) => escapeCell(c.value(row))).join(" | ")} |`;
  });
  return [header, sep, ...body].join("\n");
}

function summarize(rows, getLicense) {
  const counts = new Map();
  for (const row of rows) {
    const key = getLicense(row) || "UNKNOWN";
    counts.set(key, (counts.get(key) || 0) + 1);
  }
  return [...counts.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([license, count]) => ({ license, count }));
}

const goModules = parseJSONStream(fs.readFileSync(path.join(tmpDir, "go-modules.jsonstream"), "utf8"))
  .filter((mod) => !mod.Main)
  .map((mod) => ({
    name: mod.Path,
    version: mod.Version || "",
    license: detectGoLicense(mod),
    source: moduleSource(mod.Path),
  }))
  .sort((a, b) => a.name.localeCompare(b.name));

const webLicenseJSON = JSON.parse(fs.readFileSync(path.join(tmpDir, "web-licenses.json"), "utf8"));
const webDeps = [];

for (const [license, packages] of Object.entries(webLicenseJSON)) {
  for (const pkg of packages) {
    webDeps.push({
      name: pkg.name,
      version: (pkg.versions || []).join(", "),
      license,
      source: pkg.homepage || "",
    });
  }
}
webDeps.sort((a, b) => a.name.localeCompare(b.name));

const generatedAt = new Date().toISOString().slice(0, 10);
const lines = [];

lines.push("# Third-Party Notices");
lines.push("");
lines.push(`Generated on ${generatedAt} from the current dependency lock state.`);
lines.push("");
lines.push("This file summarizes third-party dependencies used by AI Proxy release artifacts. It is not legal advice. Preserve this file together with distributed binaries and container images, along with the root LICENSE file.");
lines.push("");
lines.push("## Coverage");
lines.push("");
lines.push("- Go modules are collected with `go list -m -json all` from `core`, which includes the workspace modules used by the backend release build.");
lines.push("- Web production dependencies are collected with `pnpm licenses list --prod --json` from `web`.");
lines.push("- Go license identifiers are best-effort detections from module LICENSE/COPYING/NOTICE files in the local module cache. Entries marked `UNKNOWN` need manual review before a formal compliance sign-off.");
lines.push("");
lines.push("## License Summary");
lines.push("");
lines.push("### Go Modules");
lines.push("");
lines.push(table(summarize(goModules, (row) => row.license), [
  { label: "License", value: (row) => row.license },
  { label: "Count", value: (row) => row.count },
]));
lines.push("");
lines.push("### Web Production Dependencies");
lines.push("");
lines.push(table(summarize(webDeps, (row) => row.license), [
  { label: "License", value: (row) => row.license },
  { label: "Count", value: (row) => row.count },
]));
lines.push("");
lines.push("## Go Modules");
lines.push("");
lines.push(table(goModules, [
  { label: "Module", value: (row) => row.name },
  { label: "Version", value: (row) => row.version },
  { label: "Detected License", value: (row) => row.license },
  { label: "Source", value: (row) => row.source },
]));
lines.push("");
lines.push("## Web Production Dependencies");
lines.push("");
lines.push(table(webDeps, [
  { label: "Package", value: (row) => row.name },
  { label: "Version", value: (row) => row.version },
  { label: "License", value: (row) => row.license },
  { label: "Homepage", value: (row) => row.source },
]));
lines.push("");

fs.writeFileSync(output, `${lines.join("\n")}\n`);
NODE

echo "Generated ${OUTPUT}"
