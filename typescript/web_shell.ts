// An already finished command
export interface CommandOut {
    Dir: string,
    Stdout: string,
    Stderr: string,
    Err: error
}

interface error {
    Text: string
}

export interface Log {
    (command: string, output: CommandOut): void
}

// TODO: move to simple_prompt
//document.addEventListener('keydown', function (e) { // Ctrl keys seem to only work with keydown
//    // TODO: Only if currently running a command
//    if (e.key === 'c' && e.ctrlKey) {
//        cancel()
//    }
//})

export async function submit(command: string) {
    if (!command) {
        return
    }
    let json;
    try {
        const resp = await fetch("/run", {
            method: "POST",
            body: command,
        });
        json = await resp.json();
    } catch (error) {
        console.error(error);
        return
    }
    return json
}

export async function cancel() {
    await fetch("/cancel", {
        method: "POST",
    });
}

export async function complete(command: string, position: number) {
    let json;
    try {
        const resp = await fetch("/complete", {
            method: "POST",
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({"text": command, "pos": position}),
        });
        json = await resp.json();
    } catch (error) {
        console.error(error);
        return
    }
    return json;
}
