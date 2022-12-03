import { Controller } from '@hotwired/stimulus';
import intlTelInput from 'intl-tel-input';
import 'intl-tel-input/build/js/utils';

export default class extends Controller {
  static targets = ['field'];

  fieldTarget!: HTMLInputElement;
  telInput: any;

  initialize() {
    this.telInput = intlTelInput(this.fieldTarget, {
      hiddenInput: 'phone_number',
      nationalMode: true,
      preferredCountries: ['jp'],
      separateDialCode: true,
    });
  }
}
