import {EditorState} from "@codemirror/state"
import {EditorView,keymap} from "@codemirror/view"
import {basicSetup} from "codemirror"
import {defaultKeymap,insertNewline} from "@codemirror/commands"
import {shell} from "@codemirror/legacy-modes/mode/shell"
import {StreamLanguage,
  defaultHighlightStyle,
  syntaxHighlighting} from "@codemirror/language"

import {CompletionContext, autocompletion, moveCompletionSelection} from "@codemirror/autocomplete"
import {cancelComplete, complete, submit} from "./web_shell.js"

function handleSubmit(view, log) {
    const command = view.state.doc.toString()
    if (!command) {
        return false
    }

    readonly = true
    submit(command).then((out) => {
        view.dispatch({ changes: {from:0, to: view.state.doc.toString().length, insert:''}})
        readonly = false
        log(command, out);
    })
    return true
}

async function shellComplete(context) {
    context.addEventListener("abort", cancelComplete)
    const command = context.state.doc.toString()
    const pos = context.pos
    if (command.length == 0) {
        return null
    }
    return complete(command, pos)
}

let readonly = false

export function createCodemirror(log, el) {
    let webshellCompletion = autocompletion({
        override: [shellComplete],
    })

    // handle Enter key
    const runCommand =
        {key: "Enter", run: (v) => { return handleSubmit(v, log)}, shift: insertNewline }
    // handle Tab key
    const runComplete =
        {
        key: "Tab",
        run: (v) => { return moveCompletionSelection(true)(v)},
        shift: (v) => { return moveCompletionSelection(false)(v)},
        preventDefault: true
    }

    let startState = EditorState.create({
        doc: "",
        extensions: [
            keymap.of([runCommand, runComplete]),
            keymap.of(defaultKeymap),
            // completion function
            webshellCompletion,
            EditorState.readOnly.of(readonly),
            StreamLanguage.define(shell),
            basicSetup,
            syntaxHighlighting(defaultHighlightStyle, {fallback: true}),
        ]
    })

    let view = new EditorView({
        state: startState,
        parent: el
    })
    let runButton = document.createElement("button")
    runButton.setAttribute("type", "submit")
    runButton.setAttribute("id", "RunButton")
    runButton.innerHTML = "Run";
    el.append(runButton)
    document.addEventListener("click", (event)=>{
        event.preventDefault();
        const target = event.target.closest("#RunButton"); // Or any other selector.
        if (target) {
            return handleSubmit(view, log)
        }
    });
    document.addEventListener("readystatechange", () => { view.focus() })
    return view
}
