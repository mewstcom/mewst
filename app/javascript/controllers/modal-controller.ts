import { Controller } from '@hotwired/stimulus';

export default class extends Controller<HTMLDialogElement> {
  connect() {
    this.element.showModal();
  }

  close(event: CustomEvent) {
    if (event.detail.success) {
      this.element.close();
    }
  }
}
