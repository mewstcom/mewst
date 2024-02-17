import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['alert', 'icon', 'message'];
  static values = {
    type: String,
    messageHtml: String,
  };

  declare readonly messageHtmlValue: string;
  declare readonly typeValue: string;

  declare readonly alertTarget: HTMLDivElement;
  declare readonly iconTarget: HTMLSpanElement;
  declare readonly messageTarget: HTMLSpanElement;

  messageHtml!: string;
  type!: string;

  connect() {
    this.type = this.typeValue || 'notice';
    this.messageHtml = this.messageHtmlValue || '';

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
    this.alertTarget.classList.add(this.alertBgClass);
    this.iconTarget.innerHTML = this.alertIconHtml;
    this.messageTarget.innerHTML = this.messageHtml;
    this.element.classList.remove('hidden');
  }

  hideFlashToast() {
    this.element.classList.add('scale-90', 'opacity-0');

    setTimeout(() => {
      this.element.classList.add('hidden');
    }, 150);
  }

  private get alertBgClass() {
    switch (this.type) {
      case 'alert':
        return 'alert-error';
      case 'notice':
        return 'alert-success';
      default:
        return 'alert-info';
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
