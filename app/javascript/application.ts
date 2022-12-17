import { Application } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';
import ujs from '@rails/ujs';
import 'bootstrap';

import AutosizeController from './controllers/autosize-controller';
import ComponentDataFetcherController from './controllers/component-data-fetcher-controller';
import FollowButtonController from './controllers/follow-button-controller';
import PhoneNumberInputController from "./controllers/phone-number-input-controller";
import PostCardController from "./controllers/post-card-controller";
import StickyPostController from './controllers/sticky-post-controller';

window.Stimulus = Application.start();
Stimulus.register('autosize', AutosizeController);
Stimulus.register('component-data-fetcher', ComponentDataFetcherController);
Stimulus.register('follow-button', FollowButtonController);
Stimulus.register('phone-number-input', PhoneNumberInputController);
Stimulus.register('post-card', PostCardController);
Stimulus.register('sticky-post', StickyPostController);

ujs.start();
Turbo.start();
