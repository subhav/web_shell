#!/usr/bin/env node

import esbuildServe from "esbuild-serve";

esbuildServe(
  {
    logLevel: "info",
    entryPoints: ["typescript/codemirror.ts"],
    bundle: true,
    outfile: "web/assets/codemirror.js",
  },
  { root: "web" }
);
