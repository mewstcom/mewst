import { Controller } from '@hotwired/stimulus';

export default class extends Controller<HTMLDialogElement> {
  connect() {
    this.element.showModal();
  }

  close(event: CustomEvent) {
    // モーダルの中でリンクカードを追加したときモーダルを閉じないようにする
    if (event.target.dataset.action !== 'turbo:submit-end->modal#close') {
      return;
    }

    if (event.detail.success) {
      this.element.close();
    }
  }
}
