import { Controller } from '@hotwired/stimulus';
import { useClickOutside } from 'stimulus-use';

export default class extends Controller<HTMLDetailsElement> {
  static targets = ['button', 'list'];

  declare readonly buttonTarget: HTMLElement;
  declare readonly listTarget: HTMLUListElement;

  connect() {
    useClickOutside(this, { element: this.buttonTarget });
  }

  clickOutside(_event) {
    this.element.removeAttribute('open');
  }

  toggle(event) {
    event.preventDefault();
    this.element.toggleAttribute('open');
  }

  close(event) {
    event.preventDefault();
    this.element.removeAttribute('open');
  }
}
