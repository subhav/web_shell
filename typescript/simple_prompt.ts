import {complete, submit, Log} from "./web_shell"

async function handleSubmit(commandEl: HTMLTextAreaElement, log: Log) {
    event.preventDefault()
    const command = commandEl.value
    if (!command) {
        return
    }
    const listEl = document.getElementById("completion-list")
    listEl.innerHTML = ''

    commandEl.value = ""
    commandEl.disabled = true;
    submit(command).then((out) => {
        log(command, out)
        commandEl.disabled = false
        commandEl.focus()
    })
}

export function createSimplePrompt(log: Log, e: HTMLDivElement) {
    // Prompt
    let prompt = document.createElement("textarea")
    prompt.setAttribute("id", "command")
    prompt.setAttribute("name", "command")
    e.append(prompt)

    // Run button
    let runButton = document.createElement("button")
    runButton.setAttribute("type", "submit")
    runButton.setAttribute("id", "RunButton")
    runButton.innerHTML = "Run";
    e.append(runButton)
    // Completion
    let completionDiv = document.createElement("div")
    let completionUl = document.createElement("ul")
    completionDiv.setAttribute("id", "completion")
    completionUl.setAttribute("id", "completion-list")
    completionDiv.append(completionUl)
    e.append(completionDiv)

    document.addEventListener("click", (event)=>{
        event.preventDefault();
        const target = event.target.closest("#RunButton")
        if (target) {
            return handleSubmit(prompt, log)
        }
    })
    document.addEventListener('keydown', function (e) {
        const target = event.target.closest("#command")
        if (target) {
            if (e.key === 'Enter' && !e.shiftKey) {
                handleSubmit(prompt, log);
            }
            if (e.key === 'Tab' && !e.shiftKey) {
                e.preventDefault()
                handleComplete(e, prompt);
            }
        }
    })

}

function updateLogElement(wrapper, resp) {
    wrapper.open = true;
    if (resp.Err) {
        wrapper.className = "failed";
    }

    function addHtml(className, value) {
        if (value) {
            let div = document.createElement("div");
            div.classList.add("term-container");
            div.classList.add(className);
            div.innerHTML = value;
            wrapper.append(div);
        }
    }
    function addPre(className, value) {
        if (value) {
            let pre = document.createElement("pre");
            pre.className = className;
            pre.textContent = value;
            wrapper.append(pre);
        }
    }
    addHtml("stdout", resp.Stdout)
    addPre("stderr", resp.Stderr)
    addPre("err", resp.Err && resp.Err.Text)
    console.log(resp);
}

async function handleComplete(event, commandEl: HTMLTextAreaElement) {
    event.preventDefault();
    const command = commandEl.value;
    let words = command.split(" ")
    const lastWord = words[words.length - 1]
    if (!command) {
        return
    }
    let listEl = document.getElementById("completion-list");
    if (!listEl) {
        return
    }

    let curComp = document.getElementsByClassName("current-completion")[0];
    if (curComp && curComp.innerHTML === lastWord) {
        let nextComp = curComp.nextElementSibling;
        if (nextComp == null) {
            nextComp = listEl.firstElementChild;
            if (nextComp == null) {
                return
            }
        }
        curComp.classList.remove("current-completion");
        nextComp.classList.add("current-completion");
        words[words.length -1] = nextComp.innerHTML;
        commandEl.value = words.join(" ");
        return;
    }

    listEl.innerHTML = '';
    complete(command, command.length).then((json) => {
        let completions = json.options.map((o) => { return o.label })
        updateCompletionList(listEl, [ lastWord, ...completions]);
    })
}

function updateCompletionList(listEl, resp) {
    for (const i in resp) {
        let li = document.createElement("li");
        li.classList.add("completion-element");
        li.innerHTML = resp[i];
        listEl.append(li);
    }
    // First element is what's in the prompt
    listEl.firstElementChild.hidden = true;
    listEl.firstElementChild.classList.add("current-completion")
}
