import "basecoat-css/all";

import { setupBackLinks } from "./back_link";

// フォーム内の全送信ボタンを無効化し、二重送信を防止する
window.disableSubmitButtons = (form) => {
  form.querySelectorAll('button[type="submit"]').forEach((btn) => {
    btn.disabled = true;
  });
};

// Character counter: shows the remaining character count next to a textarea.
// Wire it with <span data-character-counter-for="<textarea id>"
// data-character-counter-max="160">. Counts Unicode code points to match the
// server-side validation (Go counts runes), unlike the Rails counter which
// counts grapheme clusters.
//
// [Ja] 文字数カウンター: textarea の残り文字数を表示する。
// <span data-character-counter-for="<textarea の id>"
// data-character-counter-max="160"> で配線する。サーバー側バリデーション
// (Go は rune 数) に合わせて Unicode コードポイント数で数える (Rails の
// カウンターは書記素クラスター数で数えるが、ここでは揃えない)。
const setupCharacterCounters = () => {
  document.querySelectorAll("[data-character-counter-for]").forEach((counter) => {
    const textarea = document.getElementById(counter.dataset.characterCounterFor);
    if (!textarea) return;

    const max = Number(counter.dataset.characterCounterMax);
    const update = () => {
      // Count a newline as a single code point. The form submits CRLF, but the
      // server folds it back to LF before validating/storing, so the counter
      // matches the server by counting the textarea value (LF) as-is.
      // [Ja] 改行は 1 コードポイントとして数える。フォーム送信時は CRLF だが、
      // サーバーが検証・保存の前に LF へ畳み戻すため、textarea の値 (LF) を
      // そのまま数えればサーバーと一致する。
      counter.textContent = String(max - [...textarea.value].length);
    };

    textarea.addEventListener("input", update);
    update();
  });
};

// Autosize: grows a <textarea data-autosize> to fit its content on input.
// Also runs once at setup so a re-rendered form (e.g. after a validation
// error) starts at the right height.
//
// [Ja] autosize: <textarea data-autosize> を入力に応じて内容の高さに合わせて
// 伸ばす。再描画されたフォーム (バリデーションエラー後など) が最初から適切な
// 高さになるよう、セットアップ時にも 1 回実行する。
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

// Link card URL detection: when a URL appears in a
// <textarea data-link-card-path="/links/new">, fetch the add-link-card prompt
// fragment into #link-form via htmx (the Rails link-card-form Stimulus
// controller equivalent). Detection pauses while #link-form has content and
// resumes once it is cleared (e.g. by the link card remove button), observed
// through a MutationObserver so this module owns the reset, not the button.
//
// [Ja] リンクカード URL 検出: <textarea data-link-card-path="/links/new"> に
// URL が現れたら、htmx でリンクカード追加プロンプトのフラグメントを #link-form に
// 取得する (Rails の link-card-form Stimulus コントローラ相当)。#link-form に
// 中身がある間は検出を停止し、空になったら (リンクカードの削除ボタンなどで)
// 再開する。空になったことは MutationObserver で監視し、リセットの責務を削除
// ボタン側ではなく本モジュールに持たせる。
const setupLinkCardDetection = () => {
  document.querySelectorAll("textarea[data-link-card-path]").forEach((textarea) => {
    const linkForm = document.getElementById("link-form");
    if (!linkForm) return;

    // Guards against duplicate requests while a fetched fragment is in flight
    // (the container is still empty until the response is swapped in).
    // [Ja] 取得したフラグメントがスワップされるまでコンテナは空のままなので、
    // その間の重複リクエストを防ぐ。
    let requested = false;

    new MutationObserver(() => {
      if (linkForm.childElementCount === 0) {
        requested = false;
      }
    }).observe(linkForm, { childList: true });

    textarea.addEventListener("input", () => {
      if (requested || linkForm.childElementCount > 0) return;

      // Detect the last URL in the body, mirroring the Rails extractLastUrl.
      // [Ja] 本文中の最後の URL を検出する (Rails の extractLastUrl に対応)。
      const urls = textarea.value.match(/https?:\/\/\S+/g);
      if (!urls) return;

      requested = true;
      window.htmx
        .ajax("GET", `${textarea.dataset.linkCardPath}?url=${encodeURIComponent(urls[urls.length - 1])}`, {
          target: linkForm,
        })
        // If the fetch fails (network error etc.) nothing is swapped in and the
        // MutationObserver never fires, so release the guard here to let the
        // next input retry. The Rails Stimulus controller stays stuck in this
        // case; recovering is an intentional deviation.
        // [Ja] 取得に失敗 (ネットワークエラー等) するとスワップが起きず
        // MutationObserver も発火しないため、ここでガードを解除して次の input で
        // 再試行できるようにする。Rails の Stimulus コントローラはこの場合
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

// main.js is loaded as a module script (deferred), so the DOM is ready here.
// [Ja] main.js は module スクリプト (defer 相当) として読み込まれるため、
// この時点で DOM は構築済み。
setupCharacterCounters();
setupAutosize();
setupLinkCardDetection();
setupBackLinks();
