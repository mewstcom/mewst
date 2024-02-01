import { Controller } from '@hotwired/stimulus';

export default class extends Controller<HTMLDivElement> {
  hide() {
    this.element.style.display = 'none';
  }
}
