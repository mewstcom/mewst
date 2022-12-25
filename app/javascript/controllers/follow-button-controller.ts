import { Controller } from '@hotwired/stimulus';

import fetcher from '../utils/fetcher';

export default class extends Controller {
  static targets = ['follow', 'following', 'loading']
  static values = {
    atname: String,
  };

  followTarget!: HTMLButtonElement;
  followingTarget!: HTMLButtonElement;
  loadingTarget!: HTMLButtonElement;

  atnameValue!: string;

  isFollowing: boolean;

  initialize() {
    document.addEventListener('component-data-fetcher:follow-button:fetched', (event: any) => {
      const data = event.detail;
      this.isFollowing = !!data[this.atnameValue];

      this.isFollowing ? this.showFollowingButton() : this.showFollowButton();
    });
  }

  showFollowButton() {
    this.followTarget.hidden = false;
    this.followingTarget.hidden = true;
    this.loadingTarget.hidden = true;
  }

  showFollowingButton() {
    this.followTarget.hidden = true;
    this.followingTarget.hidden = false;
    this.loadingTarget.hidden = true;
  }

  showLoadingButton() {
    this.followTarget.hidden = true;
    this.followingTarget.hidden = true;
    this.loadingTarget.hidden = false;
  }

  async follow(_event: any) {
    this.showLoadingButton();

    try {
      await fetcher.post('/api/internal/follow', {
        atname: this.atnameValue,
      });

      this.isFollowing = true;
      this.showFollowingButton();
    } catch (err) {
      console.error(err);
    }
  }

  async unfollow(_event: any) {
    this.showLoadingButton();

    try {
      await fetcher.post('/api/internal/unfollow', {
        atname: this.atnameValue,
      });

      this.isFollowing = false;
      this.showFollowButton();
    } catch (err) {
      console.error(err);
    }
  }
}
