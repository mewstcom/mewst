import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setupPostDrafts } from "./post_draft";

const KEY = "post_draft:user-1";

// Mirror of the debounce window in post_draft.ts. It is not exported (the module
// keeps its timing private), so keep this in sync for readable waits.
//
// [Ja] post_draft.ts のデバウンス幅の写し。モジュールはタイミングを非公開にするため
// export しておらず、待機を読みやすくするためここで同期を保つ。
const DEBOUNCE_MS = 300;

// Build the compose form with a body textarea carrying the draft key. initial
// seeds the textarea value to model the server echoing a body back after a
// failed submit (non-empty) versus a fresh visit (empty).
//
// [Ja] 下書きキーを持つ本文 textarea を含む作成フォームを構築する。initial は
// textarea の初期値で、送信失敗後にサーバーが本文をエコーバックした状態 (非空) と
// 初回訪問 (空) を再現する。
function setupDom(initial = ""): HTMLTextAreaElement {
  document.body.innerHTML = `
    <form>
      <textarea data-draft-key="${KEY}" name="content">${initial}</textarea>
      <button type="submit">post</button>
    </form>
  `;
  const textarea = document.querySelector("textarea");
  if (!textarea) throw new Error("textarea not found");
  return textarea;
}

function typeInto(textarea: HTMLTextAreaElement, value: string): void {
  textarea.value = value;
  textarea.dispatchEvent(new Event("input", { bubbles: true }));
}

function submitForm(textarea: HTMLTextAreaElement): void {
  const form = textarea.closest("form");
  if (!form) throw new Error("form not found");
  form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
}

function leavePage(): void {
  window.dispatchEvent(new Event("pagehide"));
}

describe("setupPostDrafts", () => {
  // happy-dom keeps localStorage across tests in the same file, so clear it each
  // time to keep the per-test keys independent. Fake timers drive the debounce.
  //
  // [Ja] happy-dom は同一ファイル内のテスト間で localStorage を保持するため、毎回
  // クリアしてテストごとのキーを独立させる。デバウンスはフェイクタイマーで進める。
  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
    // Drop any stubbed localStorage first so the clear below runs on the real one.
    // [Ja] スタブした localStorage を先に外し、下の clear を本物に対して実行する。
    vi.unstubAllGlobals();
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("saves the body to localStorage after the debounce elapses", () => {
    const textarea = setupDom();
    setupPostDrafts();

    typeInto(textarea, "hello");
    // Nothing is written until the debounce window passes.
    // [Ja] デバウンス幅が経過するまで何も書き込まれない。
    expect(localStorage.getItem(KEY)).toBeNull();

    vi.advanceTimersByTime(DEBOUNCE_MS);

    expect(localStorage.getItem(KEY)).toBe("hello");
  });

  it("flushes a pending save on pagehide before the debounce elapses", () => {
    const textarea = setupDom();
    setupPostDrafts();

    typeInto(textarea, "latest draft");
    leavePage();

    expect(localStorage.getItem(KEY)).toBe("latest draft");
  });

  it("coalesces rapid input into a single write of the latest value", () => {
    // happy-dom's localStorage.setItem is not Storage.prototype.setItem, so spy
    // via a stubbed storage to count writes reliably.
    // [Ja] happy-dom の localStorage.setItem は Storage.prototype.setItem では
    // ないため、スタブしたストレージ経由で書き込み回数を確実に数える。
    const store = new Map<string, string>();
    const setItem = vi.fn((k: string, v: string) => {
      store.set(k, v);
    });
    vi.stubGlobal("localStorage", {
      getItem: (k: string) => store.get(k) ?? null,
      setItem,
      removeItem: (k: string) => store.delete(k),
      clear: () => store.clear(),
    });

    const textarea = setupDom();
    setupPostDrafts();

    typeInto(textarea, "a");
    vi.advanceTimersByTime(100);
    typeInto(textarea, "ab");
    vi.advanceTimersByTime(100);
    typeInto(textarea, "abc");
    vi.advanceTimersByTime(DEBOUNCE_MS);

    expect(store.get(KEY)).toBe("abc");
    expect(setItem).toHaveBeenCalledTimes(1);
  });

  it("removes the key when the body is cleared", () => {
    localStorage.setItem(KEY, "stale");
    const textarea = setupDom("stale");
    setupPostDrafts();

    typeInto(textarea, "");
    vi.advanceTimersByTime(DEBOUNCE_MS);

    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it("restores a saved draft into an empty textarea and notifies dependents via input", () => {
    localStorage.setItem(KEY, "restored draft");
    const textarea = setupDom("");
    // Register a dependent input listener (stand-in for autosize / counter /
    // link card detection) before setup, so the restore's dispatched input
    // reaches it exactly once.
    // [Ja] 依存する input リスナー (autosize / カウンター / リンクカード検出の代役) を
    // setup 前に登録し、復元が発火する input がちょうど 1 回届くことを見る。
    const dependent = vi.fn();
    textarea.addEventListener("input", dependent);

    setupPostDrafts();

    expect(textarea.value).toBe("restored draft");
    expect(dependent).toHaveBeenCalledTimes(1);
  });

  it("does not overwrite a non-empty textarea (a server echo wins over the draft)", () => {
    localStorage.setItem(KEY, "restored draft");
    const textarea = setupDom("echoed body");

    setupPostDrafts();

    expect(textarea.value).toBe("echoed body");
  });

  it("clears the draft on submit", () => {
    localStorage.setItem(KEY, "draft");
    // Seed the textarea non-empty so setup does not take the restore path.
    // [Ja] setup が復元パスに入らないよう textarea を非空でシードする。
    const textarea = setupDom("draft");
    setupPostDrafts();

    submitForm(textarea);

    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it("cancels a pending save on submit so it cannot resurrect the sent draft", () => {
    const textarea = setupDom("draft");
    setupPostDrafts();

    typeInto(textarea, "a late edit");
    submitForm(textarea);
    // Exercise both exit paths after submit; neither may restore the cleared key.
    // [Ja] submit 後の両方の離脱経路を実行し、クリア済みキーが復活しないことを見る。
    leavePage();
    vi.advanceTimersByTime(DEBOUNCE_MS);

    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it("degrades to a no-op when localStorage throws (e.g. private mode)", () => {
    // Stub the whole storage to throw on every access (private mode / disabled),
    // covering both the restore read and the debounced write.
    // [Ja] ストレージ全体をアクセスのたびに throw するようスタブし (プライベート
    // モード / 無効)、復元の読み取りとデバウンス書き込みの双方を網羅する。
    const blocked = () => {
      throw new Error("storage disabled");
    };
    vi.stubGlobal("localStorage", {
      getItem: blocked,
      setItem: blocked,
      removeItem: blocked,
      clear: () => {},
    });
    const textarea = setupDom();
    setupPostDrafts();

    expect(() => {
      typeInto(textarea, "hello");
      vi.advanceTimersByTime(DEBOUNCE_MS);
    }).not.toThrow();
  });
});
