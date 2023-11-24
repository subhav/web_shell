import { createCodemirror } from "./codemirror";
import { Log } from "./simple_log";
import { createSimplePrompt } from "./simple_prompt";

let el = document.getElementById("prompt")
createCodemirror(Log, el)
//createSimplePrompt(Log, el)
