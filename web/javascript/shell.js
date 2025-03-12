import { createCodemirror } from "./codemirror.js";
import { Log } from "./simple_log.js";
import { createSimplePrompt } from "./simple_prompt.js";

let el = document.getElementById("prompt")
createCodemirror(Log, el)
//createSimplePrompt(Log, el)
