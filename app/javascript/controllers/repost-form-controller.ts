import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static values = {
    postId: String,
  };

  postIdValue!: string;

  initialize() {
    this.element.addEventListener('turbo:submit-start', () => {
      this.dispatch("submit-start", { detail: { postId: this.postIdValue } });
    });

    this.element.addEventListener('turbo:submit-end', () => {
      console.log('submit done!', this.postIdValue);
      this.dispatch("submit-end", { detail: { postId: this.postIdValue } });
    });
  }
}
