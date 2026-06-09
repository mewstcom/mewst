// Back link: returns the user to where they came from. Wire it with
// <a data-back-link href="/home">. The href is a fallback used when there is no
// usable history entry (direct visit, external referrer, or JS disabled —
// progressive enhancement). When the referrer is same-origin, intercept the
// click and history.back() instead, returning to the exact previous page
// without polluting history. This restores the "close and return" action the
// navbar new link lost when it changed from a modal to a plain full-page link.
// The same-origin guard keeps external referrers from sending the user back
// off-site.
//
// [Ja] 戻るリンク: アクセス元のページにユーザーを戻す。
// <a data-back-link href="/home"> で配線する。href は使える履歴が無いとき
// (直接アクセス・外部からの遷移・JS 無効。プログレッシブエンハンスメント) の
// フォールバック。referrer が同一オリジンのときはクリックを横取りして
// history.back() し、履歴を汚さずに直前のページへ正確に戻る。これは navbar の
// new がモーダルからプレーンなフルページリンクに変わったことで失われた
// 「閉じて元に戻る」操作の代替。同一オリジン判定により、外部からの遷移で外部
// サイトへ戻るのを防ぐ。
export const setupBackLinks = (): void => {
  document.querySelectorAll<HTMLElement>("[data-back-link]").forEach((link) => {
    link.addEventListener("click", (event) => {
      // Let the browser handle modified / non-primary clicks (open in new
      // tab/window, etc.) — only hijack a plain left click. Otherwise a
      // Cmd/Ctrl-click meant to open the href in a new tab would be canceled
      // and history.back() the current tab instead.
      // [Ja] 修飾キー付き・主ボタン以外のクリック (新規タブ/ウィンドウで開く等)
      // はブラウザの既定動作に任せ、素の左クリックのときだけ横取りする。さもないと
      // href を新規タブで開くつもりの Cmd/Ctrl + クリックを潰し、現在タブで
      // history.back() してしまう。
      if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return;
      }

      const referrer = document.referrer;
      if (!referrer) return;

      let sameOrigin = false;
      try {
        sameOrigin = new URL(referrer).origin === location.origin;
      } catch {
        sameOrigin = false;
      }

      // Only intercept when there is a same-origin entry to go back to. A tab
      // opened fresh (middle-click / "open in new tab") has a same-origin
      // referrer but history.length === 1, so history.back() would be a no-op;
      // fall through to the href so the click is not dead.
      // [Ja] 同一オリジンの戻り先が履歴にあるときだけ横取りする。新規タブで開いた
      // 場合 (中クリック・「新しいタブで開く」) は referrer が同一オリジンでも
      // history.length === 1 で history.back() が no-op になるため、href に素通り
      // させてクリックが死なないようにする。
      if (sameOrigin && history.length > 1) {
        event.preventDefault();
        history.back();
      }
    });
  });
};
