import "basecoat-css/all";

import { setupBackLinks } from "./back_link";
import { setupPostDrafts } from "./post_draft";

// フォーム内の全送信ボタンを無効化し、二重送信を防止する
window.disableSubmitButtons = (form) => {
  form.querySelectorAll('button[type="submit"]').forEach((btn) => {
    btn.disabled = true;
  });
};

// 文字数カウンター: textareaの残り文字数を表示する。
// <span data-character-counter-for="<textareaのid>"
// data-character-counter-max="160"> で配線する。サーバー側バリデーション
// (Goはrune数) に合わせてUnicodeコードポイント数で数える (Railsの
// カウンターは書記素クラスター数で数えるが、ここでは揃えない)。
const setupCharacterCounters = () => {
  document.querySelectorAll("[data-character-counter-for]").forEach((counter) => {
    const textarea = document.getElementById(counter.dataset.characterCounterFor);
    if (!textarea) return;

    const max = Number(counter.dataset.characterCounterMax);
    const update = () => {
      // 改行は1コードポイントとして数える。フォーム送信時はCRLFだが、
      // サーバーが検証・保存の前にLFへ畳み戻すため、textareaの値 (LF) を
      // そのまま数えればサーバーと一致する。
      counter.textContent = String(max - [...textarea.value].length);
    };

    textarea.addEventListener("input", update);
    update();
  });
};

// フォーカスプロキシ: data-focus-textarea="<textareaのid>" を持つ要素を
// クリックすると、そのtextareaにフォーカスする。これにより枠付きボックス下部の
// 文字数カウンター行がtextareaの一部のように振る舞う (cursor-textクラスと
// 組み合わせてテキストカーソルも揃える)。クリックは本文の下に着地するため、
// キャレットは末尾へ移動する。
const setupTextareaFocusProxies = () => {
  document.querySelectorAll("[data-focus-textarea]").forEach((proxy) => {
    const textarea = document.getElementById(proxy.dataset.focusTextarea);
    if (!textarea) return;

    proxy.addEventListener("mousedown", (event) => {
      // 除外領域 (例: #link-form内のリンクカード) はそのままにしてクリックや
      // テキスト選択を維持し、プロキシ内のそれ以外のクリックだけtextareaへ
      // フォーカスする。
      if (event.target.closest("[data-focus-proxy-ignore]")) return;

      // mousedownでプロキシ上のテキスト選択が始まるのを防ぎ、代わりにtextarea
      // へフォーカスする (入力欄の余白をクリックしたときの挙動に揃える)。
      event.preventDefault();
      const end = textarea.value.length;
      textarea.focus();
      textarea.setSelectionRange(end, end);
    });
  });
};

// autosize: <textarea data-autosize> を入力に応じて内容の高さに合わせて
// 伸ばす。再描画されたフォーム (バリデーションエラー後など) が最初から適切な
// 高さになるよう、セットアップ時にも1回実行する。
const setupAutosize = () => {
  document.querySelectorAll("textarea[data-autosize]").forEach((textarea) => {
    const resize = () => {
      textarea.style.height = "auto";
      textarea.style.height = `${textarea.scrollHeight}px`;
    };

    textarea.addEventListener("input", resize);
    resize();
  });
};

// リンクカードURL検出: <textarea data-link-card-path="/links/new"> に
// URLが現れたら、htmxでリンクカード追加プロンプトのフラグメントを #link-formに
// 取得する (Railsのlink-card-form Stimulusコントローラ相当)。#link-formに
// 中身がある間は検出を停止し、空になったら (リンクカードの削除ボタンなどで)
// 再開する。空になったことはMutationObserverで監視し、リセットの責務を削除
// ボタン側ではなく本モジュールに持たせる。
const setupLinkCardDetection = () => {
  document.querySelectorAll("textarea[data-link-card-path]").forEach((textarea) => {
    const linkForm = document.getElementById("link-form");
    if (!linkForm) return;

    // 取得したフラグメントがスワップされるまでコンテナは空のままなので、
    // その間の重複リクエストを防ぐ。
    let requested = false;

    new MutationObserver(() => {
      if (linkForm.childElementCount === 0) {
        requested = false;
      }
    }).observe(linkForm, { childList: true });

    textarea.addEventListener("input", () => {
      if (requested || linkForm.childElementCount > 0) return;

      // 本文中の最後のURLを検出する (RailsのextractLastUrlに対応)。
      const urls = textarea.value.match(/https?:\/\/\S+/g);
      if (!urls) return;

      requested = true;
      window.htmx
        .ajax("GET", `${textarea.dataset.linkCardPath}?url=${encodeURIComponent(urls[urls.length - 1])}`, {
          target: linkForm,
        })
        // 取得に失敗 (ネットワークエラー等) するとスワップが起きず
        // MutationObserverも発火しないため、ここでガードを解除して次のinputで
        // 再試行できるようにする。RailsのStimulusコントローラはこの場合
        // 停止したままであり、復帰させるのは意図的な逸脱。
        .catch(() => {})
        .finally(() => {
          if (linkForm.childElementCount === 0) {
            requested = false;
          }
        });
    });
  });
};

// main.jsはmoduleスクリプト (defer相当) として読み込まれるため、
// この時点でDOMは構築済み。
setupCharacterCounters();
setupTextareaFocusProxies();
setupAutosize();
setupLinkCardDetection();
setupBackLinks();
// setupPostDraftsは最後に実行する。下書き復元がinputを再発火した際、上で
// 登録済みのカウンター / autosize / リンクカードのリスナーに届くようにするため。
setupPostDrafts();
