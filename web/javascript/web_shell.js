// An already finished command


export async function submit(command) {
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

let completionSignal
export async function cancelComplete() {
    if(completionSignal) {
        completionSignal.abort()
    }
}

export async function complete(command, position) {
    await cancelComplete()
    completionSignal = new AbortController()
    let json;
    try {
        const resp = await fetch("/complete", {
            method: "POST",
            signal: completionSignal.signal,
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
