// 戻るリンク: アクセス元のページにユーザーを戻す。
// <a data-back-link href="/home"> で配線する。hrefは使える履歴が無いとき
// (直接アクセス・外部からの遷移・JS無効。プログレッシブエンハンスメント) の
// フォールバック。referrerが同一オリジンのときはクリックを横取りして
// history.back() し、履歴を汚さずに直前のページへ正確に戻る。これはnavbarの
// newがモーダルからプレーンなフルページリンクに変わったことで失われた
// 「閉じて元に戻る」操作の代替。同一オリジン判定により、外部からの遷移で外部
// サイトへ戻るのを防ぐ。
export const setupBackLinks = (): void => {
  document.querySelectorAll<HTMLElement>("[data-back-link]").forEach((link) => {
    link.addEventListener("click", (event) => {
      // 修飾キー付き・主ボタン以外のクリック (新規タブ/ウィンドウで開く等)
      // はブラウザの既定動作に任せ、素の左クリックのときだけ横取りする。さもないと
      // hrefを新規タブで開くつもりのCmd/Ctrl + クリックを潰し、現在タブで
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

      // 同一オリジンの戻り先が履歴にあるときだけ横取りする。新規タブで開いた
      // 場合 (中クリック・「新しいタブで開く」) はreferrerが同一オリジンでも
      // history.length === 1でhistory.back() がno-opになるため、hrefに素通り
      // させてクリックが死なないようにする。
      if (sameOrigin && history.length > 1) {
        event.preventDefault();
        history.back();
      }
    });
  });
};
