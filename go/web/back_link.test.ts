import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setupBackLinks } from "./back_link";

// happy-dom exposes document.referrer as a prototype getter, so shadow it with an
// own accessor to feed each test a chosen referrer.
//
// [Ja] happy-dom は document.referrer をプロトタイプの getter として公開するため、
// 独自のアクセサで上書きしてテストごとに任意の referrer を差し込む。
function setReferrer(value: string): void {
  Object.defineProperty(document, "referrer", { configurable: true, get: () => value });
}

// setupBackLinks only hijacks when history.length > 1, so override the length to
// simulate a tab with (or without) a prior same-origin entry to go back to.
//
// [Ja] setupBackLinks は history.length > 1 のときだけ横取りするため、length を
// 上書きして、戻れる同一オリジンの履歴がある (または無い) タブを再現する。
function setHistoryLength(value: number): void {
  Object.defineProperty(window.history, "length", { configurable: true, get: () => value });
}

// Dispatch a click on the back link and return the event so the caller can read
// defaultPrevented. Defaults to a plain primary click; init overrides the
// modifier keys / button for the non-hijack cases.
//
// [Ja] 戻るリンクへクリックを発火し、呼び出し側が defaultPrevented を読めるよう
// イベントを返す。既定は素の主ボタンクリックで、init で修飾キー / ボタンを上書きして
// 非横取りのケースを作る。
function clickBackLink(init: MouseEventInit = {}): MouseEvent {
  const link = document.querySelector<HTMLElement>("[data-back-link]");
  if (!link) throw new Error("back link not found");
  const event = new MouseEvent("click", { bubbles: true, cancelable: true, button: 0, ...init });
  link.dispatchEvent(event);
  return event;
}

describe("setupBackLinks", () => {
  let backSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    document.body.innerHTML = `<a data-back-link href="/home">back</a>`;
    // history.back() would reset happy-dom's location, so stub it and assert the
    // call instead of exercising a real navigation.
    //
    // [Ja] history.back() は happy-dom の location をリセットしてしまうため、スタブ化して
    // 実際の遷移ではなく呼び出しの有無を検証する。
    backSpy = vi.spyOn(window.history, "back").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("intercepts a plain click and calls history.back() for a same-origin referrer with history to go back to", () => {
    setReferrer(`${location.origin}/previous`);
    setHistoryLength(2);
    setupBackLinks();

    const event = clickBackLink();

    expect(event.defaultPrevented).toBe(true);
    expect(backSpy).toHaveBeenCalledTimes(1);
  });

  it("does not intercept when there is no referrer", () => {
    setReferrer("");
    setHistoryLength(2);
    setupBackLinks();

    const event = clickBackLink();

    expect(event.defaultPrevented).toBe(false);
    expect(backSpy).not.toHaveBeenCalled();
  });

  it("does not intercept a cross-origin referrer", () => {
    setReferrer("https://external.example.com/page");
    setHistoryLength(2);
    setupBackLinks();

    const event = clickBackLink();

    expect(event.defaultPrevented).toBe(false);
    expect(backSpy).not.toHaveBeenCalled();
  });

  it("does not intercept when history holds only the current entry", () => {
    setReferrer(`${location.origin}/previous`);
    setHistoryLength(1);
    setupBackLinks();

    const event = clickBackLink();

    expect(event.defaultPrevented).toBe(false);
    expect(backSpy).not.toHaveBeenCalled();
  });

  // Modified / non-primary clicks (open in new tab/window etc.) must fall through
  // to the browser so the href opens; hijacking them would history.back() the
  // current tab instead.
  //
  // [Ja] 修飾キー付き / 主ボタン以外のクリック (新規タブ / ウィンドウで開く等) は
  // href が開くようブラウザに素通りさせる。横取りすると代わりに現在タブで
  // history.back() してしまう。
  const nonHijackClicks: Array<{ name: string; init: MouseEventInit }> = [
    { name: "metaKey", init: { metaKey: true } },
    { name: "ctrlKey", init: { ctrlKey: true } },
    { name: "shiftKey", init: { shiftKey: true } },
    { name: "altKey", init: { altKey: true } },
    { name: "non-primary button", init: { button: 1 } },
  ];

  it.each(nonHijackClicks)("does not intercept a $name click", ({ init }) => {
    setReferrer(`${location.origin}/previous`);
    setHistoryLength(2);
    setupBackLinks();

    const event = clickBackLink(init);

    expect(event.defaultPrevented).toBe(false);
    expect(backSpy).not.toHaveBeenCalled();
  });
});
