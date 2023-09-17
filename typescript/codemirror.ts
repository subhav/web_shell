import {EditorState} from "@codemirror/state"
import {EditorView,keymap} from "@codemirror/view"
import {basicSetup} from "codemirror"
import {defaultKeymap} from "@codemirror/commands"
import {shell} from "@codemirror/legacy-modes/mode/shell"
import {StreamLanguage,
  defaultHighlightStyle,
  syntaxHighlighting} from "@codemirror/language"

import {CompletionContext, CompletionResult, autocompletion} from "@codemirror/autocomplete"

function myCompletions(context: CompletionContext) {
  console.log(context)
  let word = context.matchBefore(/\w*/)
  if (word.from == word.to && !context.explicit)
    return null
  return {
    from: word.from,
    options: [
      {label: "match", type: "keyword"},
      {label: "hello", type: "variable", info: "(World)"},
      {label: "magic", type: "text", apply: "⠁⭒*.✩.*⭒⠁", detail: "macro"}
    ]
  }
}

async function shellComplete(context: CompletionContext) {
    const command = context.state.doc.toString()
    const pos = context.pos
    if (command.length == 0) {
        return null
    }

    // Delete existing completions
    let json;
    try {
        const resp = await fetch("/complete", {
            method: "POST",
            headers: {
            'Content-Type': 'application/json',
            },
            body: JSON.stringify({"text": command, "pos": pos}),
        });
        json = await resp.json();
    } catch (error) {
        console.error(error);
        return
    }
    console.log(json)
    return json[0]
}

let myAuto = autocompletion({
    override: [shellComplete],
  })

let startState = EditorState.create({
  doc: "Hello World",
  extensions: [
      keymap.of(defaultKeymap),
      // completion function
      myAuto,
      StreamLanguage.define(shell),
      basicSetup,
      syntaxHighlighting(defaultHighlightStyle, {fallback: true}),
  ]
})

let view = new EditorView({
  state: startState,
  parent: document.body
})
