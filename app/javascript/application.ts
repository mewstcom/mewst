import { Application } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';
import ujs from '@rails/ujs';
import 'bootstrap';

import ComponentDataFetcherController from './controllers/component-data-fetcher-controller';
import FollowButtonController from './controllers/follow-button-controller';

window.Stimulus = Application.start();
Stimulus.register('component-data-fetcher', ComponentDataFetcherController);
Stimulus.register('follow-button', FollowButtonController);

ujs.start();
Turbo.start();
