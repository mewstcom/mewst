import { Controller } from '@hotwired/stimulus';

import fetcher from '../utils/fetcher';

export default class extends Controller {
  initialize() {
    console.log('＼(^o^)／');
  }

  reload() {
    console.log('Reload!');
  }
}
