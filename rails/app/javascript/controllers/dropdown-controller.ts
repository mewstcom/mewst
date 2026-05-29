import { Controller } from "@hotwired/stimulus";
import { useClickOutside } from "stimulus-use";

export default class extends Controller<HTMLDetailsElement> {
  static targets = ["list"];

  declare readonly listTarget: HTMLUListElement;

  connect() {
    useClickOutside(this);
  }

  clickOutside() {
    this.element.removeAttribute("open");
  }

  toggle(event: Event) {
    event.preventDefault();
    this.element.toggleAttribute("open");
  }

  close(event: Event) {
    event.preventDefault();
    this.element.removeAttribute("open");
  }
}
