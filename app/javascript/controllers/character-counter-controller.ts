import { Controller } from "@hotwired/stimulus";
import split = require("graphemesplit");

export default class extends Controller {
  static targets = ["counter", "textarea"];
  static values = {
    max: Number,
  };

  declare readonly counterTarget: HTMLSpanElement;
  declare readonly textareaTarget: HTMLTextAreaElement;
  declare readonly maxValue: number;

  connect() {
    this.counterTarget.innerText = `${this.maxValue}`;
  }

  updateCounter() {
    const charactersCount = split(this.textareaTarget.value).length;
    this.counterTarget.innerText = `${this.maxValue - charactersCount}`;
  }
}
