#!/usr/bin/env bash
#
# browse.sh drives playwright-cli for browser verification of the dev site:
# it generates the Basic-auth config, runs the single-step dev sign-in, reuses
# the logged-in session for screenshots, and cleans up.
#
# It expects KORYLUS_BROWSING_* in the environment, so run it under the op-run
# wrapper (see the browse-* targets in go/Makefile). Reading credentials through
# op-run avoids evaluating the .env in a shell, which would corrupt any
# credential containing a `$`. The dev server must run with Turnstile disabled
# (MEWST_TURNSTILE_DISABLE=true in the dev .env) or bot verification blocks the
# sign-in submit.
#
# [Ja] browse.sh は playwright-cli を駆動して dev サイトのブラウザ確認を行う。
# Basic 認証 config の生成・単一ステップの dev サインイン・ログイン済み
# セッションでのスクショ・後片付けをまとめる。
#
# KORYLUS_BROWSING_* が環境にある前提なので、op run ラッパー配下 (go/Makefile の
# browse-* ターゲット) から実行する。creds を op run 経由で読むことで、.env を
# シェル評価して `$` を含む creds を壊すのを避ける。dev サーバは Turnstile を
# 無効化 (dev の .env で MEWST_TURNSTILE_DISABLE=true) して起動している必要が
# あり、でないと Bot 検証でサインインの送信が弾かれる。
set -euo pipefail

SESSION=dev
TMP_DIR=/workspace/tmp
CONFIG_FILE="$TMP_DIR/browse-cli.config.json"
ORIGIN_FILE="$TMP_DIR/browse-cli.origin"
PROFILE_DIR="$TMP_DIR/browse-cli-profile"
SHOT_DIR="$TMP_DIR/browse"

pw() { playwright-cli -s="$SESSION" "$@"; }

# pw_checked handles command-level Playwright errors, which playwright-cli
# reports in its output while still exiting with status 0. Callers may discard
# successful output, but errors are always preserved on stderr.
#
# [Ja] pw_checked は、playwright-cli が終了コード 0 のまま出力へ記録する
# Playwright のコマンドレベルエラーを検出する。呼び出し側は成功時の出力を
# 捨てられるが、エラーは常に stderr へ残す。
pw_checked() {
  local output
  if ! output="$(pw "$@" 2>&1)"; then
    printf '%s\n' "$output" >&2
    return 1
  fi
  if [[ "$output" == *"### Error"* ]]; then
    printf '%s\n' "$output" >&2
    return 1
  fi
  printf '%s\n' "$output"
}

# build_config writes the Basic-auth config (httpCredentials) parsed from
# KORYLUS_BROWSING_BASE_URL, plus a credential-free origin file for later
# navigation. The config carries the credentials, so it is written 0600 under
# the gitignored tmp dir and removed as soon as login captures it in the
# browser context.
#
# [Ja] build_config は KORYLUS_BROWSING_BASE_URL から Basic 認証 config
# (httpCredentials) を生成し、以降の遷移用に creds を抜いた origin ファイルも書く。
# config は creds を含むため gitignore 済み tmp に 0600 で書き、ログインが
# ブラウザコンテキストに取り込んだ直後に削除する。
build_config() {
  mkdir -p "$TMP_DIR"
  node -e '
    const fs = require("fs");
    const raw = process.env.KORYLUS_BROWSING_BASE_URL || "";
    if (!raw) { console.error("KORYLUS_BROWSING_BASE_URL is not set"); process.exit(1); }
    const u = new URL(raw);
    const cfg = { browser: { contextOptions: { httpCredentials: {
      username: decodeURIComponent(u.username),
      password: decodeURIComponent(u.password),
    } } } };
    fs.writeFileSync(process.argv[1], JSON.stringify(cfg), { mode: 0o600 });
    u.username = u.password = "";
    fs.writeFileSync(process.argv[2], u.origin);
  ' "$CONFIG_FILE" "$ORIGIN_FILE"
}

cmd_login() {
  local n="${1:-1}"
  local email_var="KORYLUS_BROWSING_USER${n}_EMAIL"
  local pass_var="KORYLUS_BROWSING_USER${n}_PASSWORD"
  local email="${!email_var:-}"
  local pass="${!pass_var:-}"
  if [ -z "$email" ] || [ -z "$pass" ]; then
    echo "USER${n} credentials are not set (${email_var} / ${pass_var})" >&2
    exit 1
  fi

  # Remove the credential-bearing config on any exit, so a mid-login failure
  # (a set -e abort before the explicit rm below) never leaves credentials at
  # rest.
  #
  # [Ja] creds を含む config をどの終了経路でも削除し、ログイン途中の失敗
  # (下の明示 rm へ到達する前の set -e abort) でも creds をディスクに残さない。
  trap 'rm -f "$CONFIG_FILE"' EXIT

  build_config
  local origin
  origin="$(cat "$ORIGIN_FILE")"

  # Basic auth is passed via the config (httpCredentials); the persistent
  # profile keeps the login cookies on disk so a still-running session survives
  # across separate shell invocations.
  #
  # [Ja] Basic 認証は config (httpCredentials) で渡す。永続プロファイルはログイン
  # Cookie をディスクに残し、起動中のセッションが別々のシェル呼び出しをまたいで
  # 生き続けられるようにする。
  pw_checked open "$origin/sign_in" --browser=chromium --persistent --profile="$PROFILE_DIR" --config="$CONFIG_FILE" >/dev/null

  # Mewst's sign-in is a single-step form (email + password submitted together).
  # Fill email without submitting, then submit with Enter on the password field.
  # Turnstile must be disabled (MEWST_TURNSTILE_DISABLE=true) or the submit is
  # blocked. The name/attribute locators avoid depending on label text, which
  # changes with the locale.
  #
  # [Ja] Mewst のサインインは単一ステップのフォーム (email + password を一括送信)。
  # email は送信せずに入力し、password で Enter を押して送信する。Turnstile は
  # 無効化 (MEWST_TURNSTILE_DISABLE=true) されている必要があり、でないと送信が
  # 弾かれる。name / attribute ベースのロケータはラベル文言に依存せず、locale で
  # 変わらない。
  pw_checked fill 'input[name="email"]' "$email" >/dev/null
  pw_checked fill 'input[name="password"]' "$pass" --submit >/dev/null

  # The context now holds the credentials, so the on-disk config is no longer
  # needed; drop it to avoid leaving credentials at rest.
  #
  # [Ja] コンテキストが creds を保持したので、ディスク上の config はもう不要。
  # creds を残さないため削除する。
  rm -f "$CONFIG_FILE"

  # Report the post-login URL. A successful sign-in redirects away from
  # /sign_in (to home or the back URL); staying on /sign_in (a 422 form
  # re-render) means not signed in and fails the command. The pathname is
  # extracted with a regex rather than `new URL()`: playwright-cli's run-code
  # runs in a sandbox where the URL constructor is not defined. The signed-in
  # decision is returned as a sentinel and acted on in bash so a failed login
  # exits non-zero (a throw inside run-code only prints an error and exits 0).
  #
  # [Ja] ログイン後の URL を報告する。サインイン成功時は /sign_in から離れる
  # (ホームか back URL へ)。/sign_in に留まる (422 のフォーム再描画) 場合は
  # 未ログインを意味するため、コマンドを失敗させる。pathname は `new URL()` では
  # なく正規表現で取り出す。playwright-cli の run-code は URL コンストラクタが
  # 未定義のサンドボックスで動くため。ログイン可否は sentinel で返して bash 側で
  # 判定し、失敗時に非ゼロ終了させる (run-code 内の throw はエラーを表示するだけで
  # 終了コードは 0 になるため)。
  local result
  result="$(pw_checked --raw run-code "async page => {
    await page.waitForLoadState('networkidle');
    const href = page.url();
    const path = href.replace(/^[a-z][a-z0-9+.-]*:\/\/[^/]+/i, '').replace(/[?#].*/, '');
    return (path === '/sign_in' ? 'NOT_SIGNED_IN ' : 'SIGNED_IN ') + href;
  }")"

  # --raw wraps the returned string in double quotes; strip them before matching.
  #
  # [Ja] --raw は返り値の文字列を二重引用符で囲むため、判定前に取り除く。
  result="${result%\"}"
  result="${result#\"}"

  if [[ "$result" == NOT_SIGNED_IN* ]]; then
    echo "sign-in did not complete (still on /sign_in): ${result#NOT_SIGNED_IN }" >&2
    exit 1
  fi
  if [[ "$result" != "SIGNED_IN $origin" && "$result" != "SIGNED_IN $origin/"* ]]; then
    echo "could not verify sign-in at the expected origin: $result" >&2
    exit 1
  fi
  echo "logged in as USER${n}: ${result#SIGNED_IN }"
}

cmd_shot() {
  local path="${1:-/}"
  if [ ! -f "$ORIGIN_FILE" ]; then
    echo "no active session; run 'browse.sh login' first" >&2
    exit 1
  fi
  mkdir -p "$SHOT_DIR"
  local origin
  origin="$(cat "$ORIGIN_FILE")"
  local name
  name="$(printf '%s' "$path" | sed 's#[^a-zA-Z0-9]#_#g; s#^_*##')"
  [ -n "$name" ] || name=home
  local filename="$SHOT_DIR/$name.png"

  # Remove a previous screenshot before navigating so a failed capture can
  # never be mistaken for a fresh result.
  #
  # [Ja] 撮影前に同名の既存スクリーンショットを削除し、撮影失敗時に古い画像を
  # 新しい結果と誤認できないようにする。
  rm -f "$filename"

  pw_checked goto "$origin$path" >/dev/null

  local actual_url
  actual_url="$(pw_checked --raw run-code "async page => {
    await page.waitForLoadState('networkidle');
    return page.url();
  }")"
  actual_url="${actual_url%\"}"
  actual_url="${actual_url#\"}"
  if [[ "$actual_url" != "$origin" && "$actual_url" != "$origin/"* ]]; then
    echo "page left the expected origin: $actual_url" >&2
    exit 1
  fi

  pw_checked screenshot --filename="$filename" >/dev/null
  if [ ! -s "$filename" ]; then
    echo "screenshot was not created: $filename" >&2
    exit 1
  fi
  echo "screenshot: $filename"
}

cmd_close() {
  pw close >/dev/null 2>&1 || true
  rm -f "$CONFIG_FILE" "$ORIGIN_FILE"
  rm -rf "$PROFILE_DIR"
  echo "browser session closed and temp files removed"
}

case "${1:-}" in
  login)
    shift
    cmd_login "${1:-1}"
    ;;
  shot)
    shift
    cmd_shot "${1:-/}"
    ;;
  close)
    cmd_close
    ;;
  *)
    echo "usage: browse.sh {login [user_number] | shot <path> | close}" >&2
    exit 2
    ;;
esac
