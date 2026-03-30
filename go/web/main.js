import "basecoat-css/all";

// フォーム内の全送信ボタンを無効化し、二重送信を防止する
window.disableSubmitButtons = (form) => {
  form.querySelectorAll('button[type="submit"]').forEach((btn) => {
    btn.disabled = true;
  });
};
