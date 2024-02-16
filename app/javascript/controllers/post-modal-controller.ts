import { Controller } from '@hotwired/stimulus';
import * as bootstrap from 'bootstrap';

export default class extends Controller<HTMLDivElement> {
  static targets = ['content'];

  declare readonly contentTarget: HTMLTextAreaElement;

  connect() {
    const modal = new bootstrap.Modal(this.element);

    this.element.addEventListener('shown.bs.modal', () => {
      this.contentTarget.focus();
    });

    document.addEventListener('post-modal:hide', () => {
      modal.hide();
    });
  }
}
