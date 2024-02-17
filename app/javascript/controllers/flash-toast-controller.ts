import { Controller } from '@hotwired/stimulus';
// import * as bootstrap from 'bootstrap';

export default class extends Controller {
  static targets = ['icon', 'message'];
  static values = {
    type: String,
    messageHtml: String,
  };

  declare readonly messageHtmlValue: string;
  declare readonly typeValue: string;

  declare readonly iconTarget: HTMLSpanElement;
  declare readonly messageTarget: HTMLSpanElement;

  messageHtml!: string;
  type!: string;
  // toast!: bootstrap.Toast;

  connect() {
    this.type = this.typeValue || 'notice';
    this.messageHtml = this.messageHtmlValue || '';
    // this.toast = bootstrap.Toast.getOrCreateInstance(this.element);

    if (this.type && this.messageHtml) {
      this.showFlashToast();
    }

    document.addEventListener('flash-toast:show', ({ detail: { type, messageHtml } }: any) => {
      this.type = type || 'notice';
      this.messageHtml = messageHtml || '';

      this.showFlashToast();
    });
  }

  showFlashToast() {
    this.element.classList.add(this.alertBgClass);
    this.iconTarget.innerHTML = this.alertIconHtml;
    this.messageTarget.innerHTML = this.messageHtml;
    // this.toast.show();
  }

  hideFlashToast() {
    // this.toast.hide();
  }

  private get alertBgClass() {
    switch (this.type) {
      case 'alert':
        return 'text-bg-danger';
      case 'notice':
        return 'text-bg-success';
      default:
        return 'text-bg-light';
    }
  }

  private get alertIconHtml() {
    switch (this.type) {
      case 'alert':
        return '<i class="bi bi-exclamation-triangle-fill"></i>';
      default:
        return '<i class="bi bi-check-circle-fill"></i>';
    }
  }
}
