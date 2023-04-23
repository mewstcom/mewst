import { Controller } from '@hotwired/stimulus';

import fetcher from '../utils/fetcher';

export default class extends Controller {
  static targets = ['button'];

  buttonTarget!: HTMLButtonElement;

  disable({ detail: { postId } }) {
    this.buttonTarget.classList.add('disabled');
  }

  reload({ detail: { postId } }) {
    console.log('Reload!');
    this.element.src = `/fragment/posts/${postId}/repost_dropdown`
  }
}
