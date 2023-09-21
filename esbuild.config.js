#!/usr/bin/env node

import esbuildServe from "esbuild-serve";

esbuildServe(
  {
    logLevel: "info",
    entryPoints: ["typescript/shell.ts"],
    bundle: true,
    outfile: "web/assets/shell.js",
  },
  { root: "web" }
);
