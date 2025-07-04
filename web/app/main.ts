import htmx, { HtmxBeforeSwapDetails, HtmxRequestConfig } from 'htmx.org';
import 'htmx-ext-response-targets';

// ============================================================================

// Define the custom event detail interface for 'htmx:afterSettle'
interface HtmxAfterSettleDetail {
    elt: HTMLElement;
    xhr: XMLHttpRequest;
    target: HTMLElement;
    requestConfig: HtmxRequestConfig;
}

// Define the custom event type for 'htmx:afterSettle'
interface HtmxAfterSettleEvent extends Event {
    detail?: HtmxAfterSettleDetail;
}

// ============================================================================

// type HtmxBeforeSwapEvent = CustomEvent<HtmxBeforeSwapDetail>;
interface HtmxBeforeSwapEvent extends Event {
    detail?: HtmxBeforeSwapDetails;
}

// ============================================================================

// IIFE (Immediately Invoked Function Expression) / Self-Executing Anonymous Function
(function () {
    type State = {
        letters: Array<string>
        inputs: NodeListOf<HTMLInputElement>
    }

    function main(): void {
        initalThemeHandler();
        document.addEventListener('DOMContentLoaded', function() {
            const inputs: NodeListOf<HTMLInputElement> =
                document.querySelectorAll(".focusable");

            const state: State = {
                letters: [],
                inputs: inputs,
            };

            themeButtonToggleHandler();
            initKeyListener(state);
            document.addEventListener('htmx:afterSettle', (event: HtmxAfterSettleEvent) => {reset(state, event)}, false);
            document.addEventListener('htmx:afterSettle', (event: HtmxAfterSettleEvent) => {onMessages(event)}, false);


            document.body.addEventListener('htmx:beforeSwap', function(evt: HtmxBeforeSwapEvent) {
                if (evt.detail?.xhr.status === 422) {
                    // allow 422 responses to swap as we are using this as a signal that
                    // a form was submitted with bad data and want to rerender with the
                    // errors
                    //
                    // set isError to false to avoid error logging in console
                    evt.detail.shouldSwap = true;
                    evt.detail.isError = true;
                    //evt.detail.target = "messages";
                }
            });
        }, false);

        // document.body.addEventListener('htmx:load', function(evt) {
        //     myJavascriptLib.init(evt.detail.elt);
        // });
    }

    function reset(state: State, event: HtmxAfterSettleEvent): void {
        if ((event?.detail?.xhr?.status ?? 200) === 422) {
            return;
        }

        state.letters = [];
        state.inputs = document.querySelectorAll(".focusable");
    }

    function onMessages(_event: HtmxAfterSettleEvent): void {
        const msgsContainer = document.getElementById('messages');
        if (msgsContainer === null) {
            return
        }

        const intervalID = setInterval(function() {
            msgsContainer.innerHTML = '';
            msgsContainer.classList.remove("*:opacity-100");
            msgsContainer.classList.remove("*:translate-y-0");
            msgsContainer.classList.add("*:opacity-0");
            msgsContainer.classList.add("*:-translate-y-36");
            clearInterval(intervalID);
        }, 5000);
    }

    function initalThemeHandler() {
        // On page load or when changing themes, best to add inline in `head` to avoid FOUC
        if (localStorage.getItem('color-theme') === 'dark' || (!('color-theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark')
        }
    }

    function themeButtonToggleHandler(): void {
        const themeToggleDarkIcon = document.getElementById('theme-toggle-dark-icon');
        const themeToggleLightIcon = document.getElementById('theme-toggle-light-icon');
        if (themeToggleDarkIcon === null || themeToggleLightIcon === null) {
            return
        }

        // Change the icons inside the button based on previous settings
        if (localStorage.getItem('color-theme') === 'dark' || (!('color-theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            themeToggleLightIcon.classList.remove('hidden');
        } else {
            themeToggleDarkIcon.classList.remove('hidden');
        }

        const themeToggleBtn = document.getElementById('theme-toggle');
        if (themeToggleBtn === null) {
            return
        }

        themeToggleBtn.addEventListener('click', function() {
            // toggle icons inside button
            themeToggleDarkIcon.classList.toggle('hidden');
            themeToggleLightIcon.classList.toggle('hidden');

            // if set via local storage previously
            if (localStorage.getItem('color-theme')) {
                if (localStorage.getItem('color-theme') === 'light') {
                    document.documentElement.classList.add('dark');
                    localStorage.setItem('color-theme', 'dark');
                } else {
                    document.documentElement.classList.remove('dark');
                    localStorage.setItem('color-theme', 'light');
                }

            // if NOT set via local storage previously
            } else {
                if (document.documentElement.classList.contains('dark')) {
                    document.documentElement.classList.remove('dark');
                    localStorage.setItem('color-theme', 'light');
                } else {
                    document.documentElement.classList.add('dark');
                    localStorage.setItem('color-theme', 'dark');
                }
            }

        });

    }

    function initKeyListener(state: State): void {
        document.addEventListener('keyup', (e: KeyboardEvent) => {
            const isSingleKey = e.key.length === 1
            const isInAllowedKeyRange = (65 <= e.key.charCodeAt(0) && e.key.charCodeAt(0) <= 122)
            const inputRowIsFillable = state.letters.length < state.inputs.length
            if (isSingleKey && isInAllowedKeyRange && inputRowIsFillable) {
                state.letters.push(e.key);
                updateInput(state);
            }

            if (e.key === "Backspace" || e.key === "Delete") {
                state.letters.pop();
                updateInput(state);
            }

            // capture "Enter" in case focus is lost from FORM but give page global enter functionality, but also preserve original form submit
            if (e.key === "Enter") {
                const gameSolvedOrLost = state.inputs.length === 0
                if (gameSolvedOrLost) {
                    return
                }

                const form = <HTMLFormElement|null>document.querySelector("#lettr-container form")
                if (form === null) {
                    return
                }

                const ae = document.activeElement
                if ((ae === null ) || form.contains(ae)) {
                    return
                }

                htmx.trigger("#lettr-container form", "submit");
            }
        });
    }

    function updateInput(state: State): void {
        state.inputs.forEach((input: HTMLInputElement, index: number) => {
            state.inputs[index].value = state.letters[index] ?? '';

            if (state.letters.length === 0) {
                state.inputs[0].focus()
            } else if (state.letters.length === state.inputs.length) {
                state.inputs[state.letters.length-1].focus();
            } else if (state.letters.length <= state.inputs.length) {
                state.inputs[state.letters.length].focus();
            }
        });
    }

    main();

})();
