import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setupBackLinks } from "./back_link";

// happy-domはdocument.referrerをプロトタイプのgetterとして公開するため、
// 独自のアクセサで上書きしてテストごとに任意のreferrerを差し込む。
function setReferrer(value: string): void {
  Object.defineProperty(document, "referrer", { configurable: true, get: () => value });
}

// setupBackLinksはhistory.length > 1のときだけ横取りするため、lengthを
// 上書きして、戻れる同一オリジンの履歴がある (または無い) タブを再現する。
function setHistoryLength(value: number): void {
  Object.defineProperty(window.history, "length", { configurable: true, get: () => value });
}

// 戻るリンクへクリックを発火し、呼び出し側がdefaultPreventedを読めるよう
// イベントを返す。既定は素の主ボタンクリックで、initで修飾キー / ボタンを上書きして
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
    // history.back() はhappy-domのlocationをリセットしてしまうため、スタブ化して
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

  // 修飾キー付き / 主ボタン以外のクリック (新規タブ / ウィンドウで開く等) は
  // hrefが開くようブラウザに素通りさせる。横取りすると代わりに現在タブで
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
