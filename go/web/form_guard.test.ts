import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setupFormGuards } from "./form_guard";

// The guard's observable effect is that a beforeunload event gets canceled, so
// dispatch one and read defaultPrevented. cancelable is required for
// preventDefault to register.
//
// [Ja] ガードの観測可能な効果は beforeunload イベントが取り消されることなので、
// 発火して defaultPrevented を読む。preventDefault が効くには cancelable が要る。
function dispatchBeforeUnload(): Event {
  const event = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(event);
  return event;
}

function firstForm(): HTMLFormElement {
  const form = document.querySelector("form");
  if (!form) throw new Error("form not found");
  return form;
}

function dispatchInput(form: HTMLFormElement): void {
  form.dispatchEvent(new Event("input", { bubbles: true }));
}

function dispatchSubmit(form: HTMLFormElement): void {
  form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
}

describe("setupFormGuards", () => {
  // happy-dom keeps window-level listeners across tests in the same file, so a
  // beforeunload guard left registered by one test would leak into the next and
  // cancel its beforeunload. Spy on window.addEventListener (it still calls
  // through) to collect every registered guard, then detach them all afterwards.
  //
  // [Ja] happy-dom は同一ファイル内のテスト間で window レベルのリスナーを保持するため、
  // あるテストで登録したままの beforeunload ガードが次のテストへ漏れて beforeunload を
  // 取り消してしまう。window.addEventListener をスパイし (呼び出しは素通りする)、登録された
  // ガードをすべて収集して後で全て解除する。
  let addSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    document.body.innerHTML = `
      <form data-form-guard>
        <input name="body" />
        <button type="submit">save</button>
      </form>
    `;
    addSpy = vi.spyOn(window, "addEventListener");
  });

  afterEach(() => {
    for (const [type, listener] of addSpy.mock.calls) {
      if (type === "beforeunload" && typeof listener === "function") {
        window.removeEventListener("beforeunload", listener as EventListener);
      }
    }
    vi.restoreAllMocks();
  });

  it("does not guard before any input", () => {
    setupFormGuards();

    const event = dispatchBeforeUnload();

    expect(event.defaultPrevented).toBe(false);
  });

  it("registers a beforeunload guard after the first input", () => {
    setupFormGuards();
    dispatchInput(firstForm());

    const event = dispatchBeforeUnload();

    expect(event.defaultPrevented).toBe(true);
  });

  it("removes the guard on submit so a legitimate save does not prompt", () => {
    setupFormGuards();
    const form = firstForm();
    dispatchInput(form);
    dispatchSubmit(form);

    const event = dispatchBeforeUnload();

    expect(event.defaultPrevented).toBe(false);
  });

  // Registering the same guard reference is idempotent (addEventListener dedupes
  // it), so repeated input must not stack listeners: a single submit's
  // removeEventListener has to clear the guard completely.
  //
  // [Ja] 同じガード参照の登録は冪等 (addEventListener が重複を排除する) なので、入力を
  // 繰り返してもリスナーは積み重ならない。1 回の submit の removeEventListener で
  // ガードが完全に消える必要がある。
  it("does not stack guards when input fires repeatedly", () => {
    setupFormGuards();
    const form = firstForm();
    dispatchInput(form);
    dispatchInput(form);
    dispatchInput(form);
    dispatchSubmit(form);

    const event = dispatchBeforeUnload();

    expect(event.defaultPrevented).toBe(false);
  });
});
