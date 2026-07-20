// Post draft autosave persists a compose textarea to localStorage and restores
// it when the user returns. Wire it with
// <textarea data-draft-key="post_draft:<user id>">. Editing saves the body
// (debounced) into the per-user key, and pagehide flushes a pending save so tab
// close, reload, and device back keep the latest value. Clearing the body removes
// the key; returning restores the draft into an empty textarea; a real submit
// clears the key. Every localStorage access is guarded so private mode or
// disabled storage degrades to a no-op without breaking the form.
//
// [Ja] 投稿下書きの自動保存は、集中作成の textarea を localStorage に保存し、戻って
// きたときに復元する。<textarea data-draft-key="post_draft:<user id>"> で配線する。
// 編集すると本文を (デバウンスして) ユーザー別キーに保存し、pagehide で保留中の保存を
// 確定するため、タブ閉じ・リロード・端末の戻るでも最新値を保てる。本文を空にすれば
// キーを削除し、戻ってくると空の textarea に下書きを復元し、実際の送信ではキーを
// クリアする。localStorage アクセスはすべて try/catch で囲み、プライベートモードや
// 無効時はフォームの通常動作を妨げず no-op に退化させる。

// Debounce window for input-driven saves. It coalesces a burst of keystrokes;
// pagehide separately flushes any write that is still pending when the page exits.
//
// [Ja] input 駆動の保存のデバウンス幅。連続したキー入力を 1 回の書き込みにまとめ、
// ページ離脱時にまだ保留中の書き込みは pagehide で別途確定する。
const DEBOUNCE_MS = 300;

// readDraft returns the stored draft, or null when storage is unavailable /
// empty. localStorage can throw (private mode, disabled, SecurityError), so the
// access is guarded and any failure reads as "no draft".
//
// [Ja] readDraft は保存済みの下書きを返す (ストレージが使えない / 空なら null)。
// localStorage は例外を投げうる (プライベートモード・無効・SecurityError) ため
// アクセスを囲み、失敗時は「下書きなし」として扱う。
const readDraft = (key: string): string | null => {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
};

// writeDraft stores the body, or removes the key when the body is empty so a
// cleared draft does not linger. Guarded so a storage failure is a no-op.
//
// [Ja] writeDraft は本文を保存する。本文が空のときはキーを削除し、空にした下書きが
// 残らないようにする。ストレージ失敗時に no-op となるよう囲む。
const writeDraft = (key: string, value: string): void => {
  try {
    if (value === "") {
      localStorage.removeItem(key);
    } else {
      localStorage.setItem(key, value);
    }
  } catch {
    // no-op: storage unavailable.
    // [Ja] no-op: ストレージが使えない。
  }
};

// clearDraft drops the stored draft (used on submit). Guarded like the others.
//
// [Ja] clearDraft は保存済みの下書きを破棄する (送信時に使う)。他と同様に囲む。
const clearDraft = (key: string): void => {
  try {
    localStorage.removeItem(key);
  } catch {
    // no-op: storage unavailable.
    // [Ja] no-op: ストレージが使えない。
  }
};

export const setupPostDrafts = (): void => {
  document.querySelectorAll<HTMLTextAreaElement>("textarea[data-draft-key]").forEach((textarea) => {
    const key = textarea.dataset.draftKey;
    if (!key) return;

    // Restore only into an empty textarea, then notify dependents with a single
    // input event so autosize, the character counter, and link card URL
    // detection update for the restored value. A non-empty textarea means the
    // server echoed the body back after a failed submit; that echo must win over
    // the stored draft, so it is left untouched.
    //
    // [Ja] 空の textarea のときだけ復元し、input イベントを 1 回発火して autosize・
    // 文字数カウンター・リンクカード URL 検出などの依存挙動を復元値で更新する。
    // textarea が非空なのは送信失敗後にサーバーが本文をエコーバックした場合であり、
    // その値は保存下書きより優先すべきなので上書きしない。
    if (textarea.value === "") {
      const saved = readDraft(key);
      if (saved) {
        textarea.value = saved;
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }

    // Register the save listener after the restore dispatch above so restoring
    // does not immediately re-save the value it just loaded.
    //
    // [Ja] 保存リスナーは上の復元 dispatch の後に登録し、復元がたった今読み込んだ
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

    // Flush the latest edit before the page is hidden because a pending timer
    // does not survive tab close, reload, or navigation. pagehide preserves
    // BFCache eligibility and also fires for history navigation.
    //
    // [Ja] 保留中のタイマーはタブ閉じ・リロード・画面遷移を越えて動作しないため、
    // ページが隠れる前に最新の編集を確定する。pagehide は BFCache 適格性を保ち、
    // 履歴移動でも発火する。
    window.addEventListener("pagehide", flushPendingSave);

    const form = textarea.closest("form");
    if (form) {
      form.addEventListener("submit", () => {
        // Cancel a pending debounced save so it cannot re-write the key after
        // the submit clears it (which would resurrect the just-sent draft).
        //
        // [Ja] 保留中のデバウンス保存を取り消し、送信でクリアした後にキーを書き戻して
        // 送信済みの下書きが復活しないようにする。
        cancelPendingSave();
        clearDraft(key);
      });
    }
  });
};
