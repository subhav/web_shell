import { CommandOut } from "./web_shell";

document.getElementById("log").addEventListener("dblclick", function(e) {
    for (let el of e.composedPath().reverse()) {
        if (el.nodeName === "DETAILS") {
            el.open = !el.open;
            break;
        }
    }
})

export function Log(command: string, output: CommandOut) {
    let wrapper = createLogElement(command);
    wrapper.open = true;
    if (output.Err) {
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
    addHtml("stdout", output.Stdout)
    addPre("stderr", output.Stderr)
    addPre("err", output.Err && output.Err.Text)
    if (output.Dir) {
        document.getElementById("dir").textContent = output.Dir
    }
    //console.log(output);
    document.scrollingElement.scrollTop = document.scrollingElement.scrollHeight;
}

function createLogElement(command: string): HTMLDetailsElement {
    // Yuck
    const logEl = document.getElementById("log");
    let wrapperEl = document.createElement("details");
    const summaryEl = document.createElement("summary");
    let codeEl = document.createElement("code");
    logEl.append(wrapperEl);
    wrapperEl.append(summaryEl);
    summaryEl.append(codeEl);
    codeEl.textContent = command;
    return wrapperEl;
}
