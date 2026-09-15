import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setupPostDrafts } from "./post_draft";

const KEY = "post_draft:user-1";

// post_draft.tsのデバウンス幅の写し。モジュールはタイミングを非公開にするため
// exportしておらず、待機を読みやすくするためここで同期を保つ。
const DEBOUNCE_MS = 300;

// 下書きキーを持つ本文textareaを含む作成フォームを構築する。initialは
// textareaの初期値で、送信失敗後にサーバーが本文をエコーバックした状態 (非空) と
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
  // happy-domは同一ファイル内のテスト間でlocalStorageを保持するため、毎回
  // クリアしてテストごとのキーを独立させる。デバウンスはフェイクタイマーで進める。
  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
    // スタブしたlocalStorageを先に外し、下のclearを本物に対して実行する。
    vi.unstubAllGlobals();
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("saves the body to localStorage after the debounce elapses", () => {
    const textarea = setupDom();
    setupPostDrafts();

    typeInto(textarea, "hello");
    // デバウンス幅が経過するまで何も書き込まれない。
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
    // happy-domのlocalStorage.setItemはStorage.prototype.setItemでは
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
    // 依存するinputリスナー (autosize / カウンター / リンクカード検出の代役) を
    // setup前に登録し、復元が発火するinputがちょうど1回届くことを見る。
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
    // setupが復元パスに入らないようtextareaを非空でシードする。
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
    // submit後の両方の離脱経路を実行し、クリア済みキーが復活しないことを見る。
    leavePage();
    vi.advanceTimersByTime(DEBOUNCE_MS);

    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it("degrades to a no-op when localStorage throws (e.g. private mode)", () => {
    // ストレージ全体をアクセスのたびにthrowするようスタブし (プライベート
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
