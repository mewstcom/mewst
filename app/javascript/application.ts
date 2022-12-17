import { Application } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';
import ujs from '@rails/ujs';
import 'bootstrap';

import AutosizeController from './controllers/autosize-controller';
import ComponentDataFetcherController from './controllers/component-data-fetcher-controller';
import FollowButtonController from './controllers/follow-button-controller';
import PhoneNumberInputController from "./controllers/phone-number-input-controller";
import PostCardController from "./controllers/post-card-controller";
import PostFormController from "./controllers/post-form-controller";
import StickyPostController from './controllers/sticky-post-controller';
import TimelineController from "./controllers/timeline-controller";

window.Stimulus = Application.start();
Stimulus.register('autosize', AutosizeController);
Stimulus.register('component-data-fetcher', ComponentDataFetcherController);
Stimulus.register('follow-button', FollowButtonController);
Stimulus.register('phone-number-input', PhoneNumberInputController);
Stimulus.register('post-card', PostCardController);
Stimulus.register('post-form', PostFormController);
Stimulus.register('sticky-post', StickyPostController);
Stimulus.register('timeline', TimelineController);

ujs.start();
Turbo.start();
