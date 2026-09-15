#!/usr/bin/env bash
#
# browse.shはplaywright-cliを駆動してdevサイトのブラウザ確認を行う。
# Basic認証configの生成・単一ステップのdevサインイン・ログイン済み
# セッションでのスクショ・後片付けをまとめる。
#
# KORYLUS_BROWSING_BASE_URLが環境にある前提なので、op runラッパー配下
# (go/Makefileのbrowse-* ターゲット) から実行する。Basic認証のcredsをop run
# 経由で読むことで、.envをシェル評価して `$` を含むcredsを壊すのを避ける。
# サインインするアカウントは環境変数からは読まず、`mewst devcreds <role>` を通じて
# シードの名簿から取る。devサーバはTurnstileを無効化 (devの .envで
# MEWST_TURNSTILE_DISABLE=true) して起動している必要があり、でないとBot検証で
# サインインの送信が弾かれる。
set -euo pipefail

SESSION=dev

# GO_DIRはGoモジュールのルート。mewstコマンドと、そのコマンドが読む名簿は
# どちらもここからの相対で解決される。作業ディレクトリに委ねず本スクリプト自身の
# 位置から求めるのは、go/ 以外の場所から始めたログインが、そのどちらでも失敗しない
# ようにするため。
GO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# DEFAULT_ROLEは、コマンドラインが役割を指定しなかったときにサインインする
# アカウント。開発環境をおもに見るための役割になる。
DEFAULT_ROLE=main

# TMP_DIRは資格情報を含むファイルとスクリーンショットの置き場。テストでは
# 開発者のファイルへ触れないよう、テスト自身の一時ディレクトリへ向ける。
TMP_DIR="${MEWST_BROWSE_TMP_DIR:-/workspace/tmp}"

CONFIG_FILE="$TMP_DIR/browse-cli.config.json"
PASSWORD_SCRIPT_FILE="$TMP_DIR/browse-cli.password.js"
ORIGIN_FILE="$TMP_DIR/browse-cli.origin"
PROFILE_DIR="$TMP_DIR/browse-cli-profile"
SHOT_DIR="$TMP_DIR/browse"

pw() { playwright-cli -s="$SESSION" "$@"; }

# pw_checkedは、playwright-cliが終了コード0のまま出力へ記録する
# Playwrightのコマンドレベルエラーを検出する。呼び出し側は成功時の出力を
# 捨てられるが、エラーは常にstderrへ残す。
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

# pw_checked_secretはplaywright-cliの診断を残しつつ、資格情報を含む
# run-codeファイルを再掲し得る成功出力セクションを抑止する。
pw_checked_secret() {
  local output
  local status=0
  output="$(pw "$@" 2>&1)" || status=$?
  if [ "$status" -eq 0 ] && [[ "$output" != *"### Error"* ]]; then
    return 0
  fi
  if [[ "$output" == *"### "* ]]; then
    printf '%s\n' "$output" | awk '/^### /{ relay = ($0 == "### Error") } relay' >&2
  else
    printf '%s\n' "$output" >&2
  fi
  return 1
}

# build_configはKORYLUS_BROWSING_BASE_URLからBasic認証config
# (httpCredentials) を生成し、以降の遷移用にcredsを抜いたoriginファイルも書く。
# configはcredsを含むためgitignore済みtmpに0600で書き、ログインが
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

# build_password_scriptは標準入力からパスワードを読み、0600のPlaywright
# run-codeファイルへ書く。JSONエンコードにより全てのパスワード文字を保持する。
build_password_script() {
  mkdir -p "$TMP_DIR"
  node -e '
    const fs = require("fs");
    const password = fs.readFileSync(0, "utf8");
    const selector = `input[name="password"]`;
    const source = "async page => {\n" +
      "  const input = page.locator(" + JSON.stringify(selector) + ");\n" +
      "  await input.fill(" + JSON.stringify(password) + ");\n" +
      "  await input.press(\"Enter\");\n" +
      "}";
    try { fs.rmSync(process.argv[1]); } catch (error) {
      if (error.code !== "ENOENT") throw error;
    }
    fs.writeFileSync(process.argv[1], source, { mode: 0o600 });
  ' "$PASSWORD_SCRIPT_FILE"
}

cmd_login() {
  local role="${1:-$DEFAULT_ROLE}"

  # アカウントはシードの名簿から取る。bashは名簿を読めないため、mewst
  # devcredsがアドレスとパスワードを1行ずつ、それだけを出力する。環境変数では
  # なくそこから取ることで、開発用アカウントを記述するファイルが1つになり、名簿で
  # アカウントを変更したときに、その後のシード実行がもう作成しないアカウントへ
  # サインインが手を伸ばす状態にならない。パスワードを引数ではなく標準出力で受け取る
  # のは、argvがマシン上のすべてのプロセスから読めるため。
  local credentials
  credentials="$(cd "$GO_DIR" && go run ./cmd/mewst devcreds "$role")"

  local lines=()
  mapfile -t lines <<<"$credentials"
  local email="${lines[0]:-}"
  local pass="${lines[1]:-}"
  if [ "${#lines[@]}" -ne 2 ] || [ -z "$email" ] || [ -z "$pass" ]; then
    echo "no credentials for role '$role': mewst devcreds must print exactly two non-empty lines" >&2
    exit 1
  fi

  # 資格情報を含むファイルをどの終了経路でも削除し、ログイン途中の失敗でも
  # credsをディスクに残さない。
  trap 'rm -f "$CONFIG_FILE" "$PASSWORD_SCRIPT_FILE"' EXIT

  build_config
  local origin
  origin="$(cat "$ORIGIN_FILE")"

  # Basic認証はconfig (httpCredentials) で渡す。永続プロファイルはログイン
  # Cookieをディスクに残し、起動中のセッションが別々のシェル呼び出しをまたいで
  # 生き続けられるようにする。
  pw_checked open "$origin/sign_in" --browser=chromium --persistent --profile="$PROFILE_DIR" --config="$CONFIG_FILE" >/dev/null

  # Mewstのサインインは単一ステップのフォーム (email + passwordを一括送信)。
  # emailは送信せずに入力し、passwordでEnterを押して送信する。Turnstileは
  # 無効化 (MEWST_TURNSTILE_DISABLE=true) されている必要があり、でないと送信が
  # 弾かれる。name / attributeベースのロケータはラベル文言に依存せず、localeで
  # 変わらない。
  pw_checked fill 'input[name="email"]' "$email" >/dev/null
  printf '%s' "$pass" | build_password_script
  if ! pw_checked_secret run-code --filename="$PASSWORD_SCRIPT_FILE"; then
    echo "sign-in did not complete: filling the password failed" >&2
    exit 1
  fi
  rm -f "$PASSWORD_SCRIPT_FILE"

  # コンテキストがcredsを保持したので、ディスク上のconfigはもう不要。
  # credsを残さないため削除する。
  rm -f "$CONFIG_FILE"

  # ログイン後のURLを報告する。サインイン成功時は /sign_inから離れる
  # (ホームかback URLへ)。/sign_inに留まる (422のフォーム再描画) 場合は
  # 未ログインを意味するため、コマンドを失敗させる。pathnameは `new URL()` では
  # なく正規表現で取り出す。playwright-cliのrun-codeはURLコンストラクタが
  # 未定義のサンドボックスで動くため。ログイン可否はsentinelで返してbash側で
  # 判定し、失敗時に非ゼロ終了させる (run-code内のthrowはエラーを表示するだけで
  # 終了コードは0になるため)。
  local result
  result="$(pw_checked --raw run-code "async page => {
    await page.waitForLoadState('networkidle');
    const href = page.url();
    const path = href.replace(/^[a-z][a-z0-9+.-]*:\/\/[^/]+/i, '').replace(/[?#].*/, '');
    return (path === '/sign_in' ? 'NOT_SIGNED_IN ' : 'SIGNED_IN ') + href;
  }")"

  # --rawは返り値の文字列を二重引用符で囲むため、判定前に取り除く。
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
  echo "logged in as ${role}: ${result#SIGNED_IN }"
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

  # 撮影前に同名の既存スクリーンショットを削除し、撮影失敗時に古い画像を
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
  rm -f "$CONFIG_FILE" "$PASSWORD_SCRIPT_FILE" "$ORIGIN_FILE"
  rm -rf "$PROFILE_DIR"
  echo "browser session closed and temp files removed"
}

main() {
  case "${1:-}" in
    login)
      shift
      cmd_login "${1:-$DEFAULT_ROLE}"
      ;;
    shot)
      shift
      cmd_shot "${1:-/}"
      ;;
    close)
      cmd_close
      ;;
    *)
      echo "usage: browse.sh {login [role] | shot <path> | close}" >&2
      return 2
      ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
