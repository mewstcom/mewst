import { Controller } from '@hotwired/stimulus';
import { useClickOutside } from 'stimulus-use';

export default class extends Controller<HTMLDetailsElement> {
  static targets = ['list'];

  declare readonly listTarget: HTMLUListElement;

  connect() {
    useClickOutside(this);
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
