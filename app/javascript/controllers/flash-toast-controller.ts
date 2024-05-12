import { Controller } from '@hotwired/stimulus';

const animationClasses = ['scale-90', 'opacity-0'];
const hiddenClass = 'hidden';

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
    this.iconTarget.classList.add(this.alertIconColorClass);
    this.iconTarget.innerHTML = this.alertIconHtml;
    this.messageTarget.innerHTML = this.messageHtml;
    this.element.classList.remove(hiddenClass, ...animationClasses);

    if (this.type === 'notice') {
      setTimeout(() => {
        this.hideFlashToast();
      }, 2000);
    }
  }

  hideFlashToast() {
    this.element.classList.add(...animationClasses);

    setTimeout(() => {
      this.element.classList.add(hiddenClass);
    }, 150);
  }

  private get alertIconColorClass() {
    switch (this.type) {
      case 'alert':
        return 'text-error';
      case 'notice':
        return 'text-success';
      default:
        return 'text-info';
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
