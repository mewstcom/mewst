import { Controller } from '@hotwired/stimulus';
import intlTelInput from 'intl-tel-input';
import 'intl-tel-input/build/js/utils';

export default class extends Controller {
  static targets = ['field']

  fieldTarget!: HTMLInputElement;
  telInput: any;

  initialize() {
    console.log('this.fieldTarget', this.fieldTarget);
    this.telInput = intlTelInput(this.fieldTarget, {
      hiddenInput: 'phone_number_full',
      nationalMode: true,
      separateDialCode: true,
      preferredCountries: ['jp'],
      // utilsScript: "https://cdnjs.cloudflare.com/ajax/libs/intl-tel-input/17.0.8/js/utils.js"
    });
  }

  input() {
    console.log('input!!!: ', this.telInput.getNumber());
  }
}
