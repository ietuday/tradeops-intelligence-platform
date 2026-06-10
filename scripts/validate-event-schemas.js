#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");

const root = process.cwd();
const schemaRoot = path.join(root, "schemas", "events");
const mappingPath = path.join(schemaRoot, "sample-mapping.json");

let failures = 0;
let warnings = 0;
let parsed = 0;

function log(level, message) {
  console.log(`[${level}] ${message}`);
}

function walkJsonFiles(dir) {
  if (!fs.existsSync(dir)) {
    return [];
  }

  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      return walkJsonFiles(fullPath);
    }
    return entry.isFile() && entry.name.endsWith(".json") ? [fullPath] : [];
  });
}

function parseJsonFile(file) {
  try {
    parsed += 1;
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    failures += 1;
    log("FAIL", `${path.relative(root, file)} is not valid JSON: ${error.message}`);
    return undefined;
  }
}

function tryLoadAjv() {
  try {
    return require("ajv");
  } catch (_) {
    return undefined;
  }
}

if (!fs.existsSync(schemaRoot)) {
  log("FAIL", "schemas/events directory does not exist.");
  process.exit(1);
}

const schemaFiles = walkJsonFiles(schemaRoot);
const schemaByPath = new Map();
for (const file of schemaFiles) {
  const schema = parseJsonFile(file);
  if (schema) {
    schemaByPath.set(path.relative(root, file), schema);
  }
}

let mappings = {};
if (fs.existsSync(mappingPath)) {
  const parsedMapping = parseJsonFile(mappingPath);
  if (parsedMapping && typeof parsedMapping === "object" && !Array.isArray(parsedMapping)) {
    mappings = parsedMapping;
  } else {
    failures += 1;
    log("FAIL", "schemas/events/sample-mapping.json must be a JSON object.");
  }
} else {
  warnings += 1;
  log("WARN", "schemas/events/sample-mapping.json is missing; sample validation skipped.");
}

const Ajv = tryLoadAjv();
let ajv;
if (Ajv) {
  ajv = new Ajv({ allErrors: true, strict: false });
  log("PASS", "Ajv is available; validating mapped samples against schemas.");
} else {
  warnings += 1;
  log("WARN", "Ajv is not installed; parsed schemas and samples only. Install ajv to enable full validation.");
}

for (const [sampleRel, schemaRel] of Object.entries(mappings)) {
  const samplePath = path.join(root, sampleRel);
  const schemaPath = path.join(root, schemaRel);

  if (!fs.existsSync(samplePath)) {
    failures += 1;
    log("FAIL", `Mapped sample is missing: ${sampleRel}`);
    continue;
  }
  if (!fs.existsSync(schemaPath)) {
    failures += 1;
    log("FAIL", `Mapped schema is missing for ${sampleRel}: ${schemaRel}`);
    continue;
  }

  const sample = parseJsonFile(samplePath);
  const schema = schemaByPath.get(schemaRel) || parseJsonFile(schemaPath);
  if (!sample || !schema || !ajv) {
    continue;
  }

  try {
    const validate = ajv.compile(schema);
    if (!validate(sample)) {
      failures += 1;
      log("FAIL", `${sampleRel} does not match ${schemaRel}: ${ajv.errorsText(validate.errors)}`);
    }
  } catch (error) {
    failures += 1;
    log("FAIL", `Could not compile ${schemaRel}: ${error.message}`);
  }
}

if (failures > 0) {
  log("FAIL", `Event schema validation completed with ${failures} failure(s), ${warnings} warning(s), ${parsed} JSON file(s) parsed.`);
  process.exit(1);
}

log("PASS", `Event schema validation completed with ${warnings} warning(s), ${parsed} JSON file(s) parsed.`);
