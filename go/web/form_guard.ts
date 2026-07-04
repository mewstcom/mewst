// Form leave guard: warns before leaving a <form data-form-guard> that has
// unsaved edits. The first input marks the form dirty and registers a
// beforeunload handler, so the browser's native confirm dialog covers every
// leave path at once — back button, tab close, reload, and the cancel button
// (which navigates away via back_link.ts). A real submit removes the handler so
// a legitimate save does not trigger the dialog. Registering the same handler
// reference is idempotent (addEventListener dedupes it), so repeated input never
// stacks listeners and no separate dirty flag is needed. The dialog text is
// fixed by the browser; modern browsers ignore any custom message, so all paths
// show the same generic prompt.
//
// [Ja] フォーム離脱ガード: <form data-form-guard> を未保存の編集を残したまま
// 離れようとしたときに警告する。最初の入力でフォームを dirty とみなして
// beforeunload ハンドラーを登録し、ブラウザ標準の確認ダイアログで全ての離脱経路
// (戻るボタン・タブ閉じ・リロード・キャンセルボタン (back_link.ts 経由で離脱)) を
// 一括でガードする。実際の送信ではハンドラーを解除し、正当な保存で確認が出ない
// ようにする。同じハンドラー参照の登録は冪等 (addEventListener が重複を排除する)
// のため、入力を繰り返してもリスナーは積み重ならず、別途 dirty フラグも要らない。
// ダイアログの文言はブラウザが固定し、現代ブラウザはカスタム文言を無視するため、
// 全経路で同じ汎用的な確認が表示される。
export const setupFormGuards = (): void => {
  document.querySelectorAll<HTMLFormElement>("form[data-form-guard]").forEach((form) => {
    const guard = (event: BeforeUnloadEvent) => {
      // preventDefault is the modern trigger; returnValue is the legacy fallback.
      //
      // [Ja] preventDefault が現代の発火方法、returnValue は旧ブラウザ向けの
      // フォールバック。
      event.preventDefault();
      event.returnValue = "";
    };

    // Any input marks the form dirty. Re-adding the same guard reference is a
    // no-op, so this doubles as the "register once when dirty" step.
    //
    // [Ja] 入力があればフォームを dirty とみなす。同じ guard 参照の再登録は no-op
    // なので、これがそのまま「dirty になったら 1 回だけ登録する」処理を兼ねる。
    form.addEventListener("input", () => {
      window.addEventListener("beforeunload", guard);
    });

    // A real submit is a legitimate save, so drop the guard before the page
    // navigates to the response. Native constraint validation suppresses the
    // submit event while the form is invalid, so the guard stays put on a failed
    // submit and keeps protecting the still-unsaved input.
    //
    // [Ja] 実際の送信は正当な保存なので、応答へ遷移する前にガードを外す。
    // ネイティブの制約検証はフォームが不正な間 submit イベントを止めるため、
    // 送信失敗時はガードが残り、未保存の入力を守り続ける。
    form.addEventListener("submit", () => {
      window.removeEventListener("beforeunload", guard);
    });
  });
};
