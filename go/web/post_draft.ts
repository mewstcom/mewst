// 投稿下書きの自動保存は、集中作成のtextareaをlocalStorageに保存し、戻って
// きたときに復元する。<textarea data-draft-key="post_draft:<user id>"> で配線する。
// 編集すると本文を (デバウンスして) ユーザー別キーに保存し、pagehideで保留中の保存を
// 確定するため、タブ閉じ・リロード・端末の戻るでも最新値を保てる。本文を空にすれば
// キーを削除し、戻ってくると空のtextareaに下書きを復元し、実際の送信ではキーを
// クリアする。localStorageアクセスはすべてtry/catchで囲み、プライベートモードや
// 無効時はフォームの通常動作を妨げずno-opに退化させる。

// input駆動の保存のデバウンス幅。連続したキー入力を1回の書き込みにまとめ、
// ページ離脱時にまだ保留中の書き込みはpagehideで別途確定する。
const DEBOUNCE_MS = 300;

// readDraftは保存済みの下書きを返す (ストレージが使えない / 空ならnull)。
// localStorageは例外を投げうる (プライベートモード・無効・SecurityError) ため
// アクセスを囲み、失敗時は「下書きなし」として扱う。
const readDraft = (key: string): string | null => {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
};

// writeDraftは本文を保存する。本文が空のときはキーを削除し、空にした下書きが
// 残らないようにする。ストレージ失敗時にno-opとなるよう囲む。
const writeDraft = (key: string, value: string): void => {
  try {
    if (value === "") {
      localStorage.removeItem(key);
    } else {
      localStorage.setItem(key, value);
    }
  } catch {
    // no-op: ストレージが使えない。
  }
};

// clearDraftは保存済みの下書きを破棄する (送信時に使う)。他と同様に囲む。
const clearDraft = (key: string): void => {
  try {
    localStorage.removeItem(key);
  } catch {
    // no-op: ストレージが使えない。
  }
};

export const setupPostDrafts = (): void => {
  document.querySelectorAll<HTMLTextAreaElement>("textarea[data-draft-key]").forEach((textarea) => {
    const key = textarea.dataset.draftKey;
    if (!key) return;

    // 空のtextareaのときだけ復元し、inputイベントを1回発火してautosize・
    // 文字数カウンター・リンクカードURL検出などの依存挙動を復元値で更新する。
    // textareaが非空なのは送信失敗後にサーバーが本文をエコーバックした場合であり、
    // その値は保存下書きより優先すべきなので上書きしない。
    if (textarea.value === "") {
      const saved = readDraft(key);
      if (saved) {
        textarea.value = saved;
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }

    // 保存リスナーは上の復元dispatchの後に登録し、復元がたった今読み込んだ
    // 値をすぐ再保存しないようにする。
    let timer: ReturnType<typeof setTimeout> | undefined;
    const cancelPendingSave = (): void => {
      if (timer === undefined) return;
      clearTimeout(timer);
      timer = undefined;
    };
    const flushPendingSave = (): void => {
      if (timer === undefined) return;
      cancelPendingSave();
      writeDraft(key, textarea.value);
    };

    textarea.addEventListener("input", () => {
      cancelPendingSave();
      timer = setTimeout(() => {
        timer = undefined;
        writeDraft(key, textarea.value);
      }, DEBOUNCE_MS);
    });

    // 保留中のタイマーはタブ閉じ・リロード・画面遷移を越えて動作しないため、
    // ページが隠れる前に最新の編集を確定する。pagehideはBFCache適格性を保ち、
    // 履歴移動でも発火する。
    window.addEventListener("pagehide", flushPendingSave);

    const form = textarea.closest("form");
    if (form) {
      form.addEventListener("submit", () => {
        // 保留中のデバウンス保存を取り消し、送信でクリアした後にキーを書き戻して
        // 送信済みの下書きが復活しないようにする。
        cancelPendingSave();
        clearDraft(key);
      });
    }
  });
};
