import { Controller } from '@hotwired/stimulus';
import autosize from 'autosize';

export default class extends Controller {
  initialize() {
    autosize(this.element);
  }
}
