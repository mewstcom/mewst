import { Application } from '@hotwired/stimulus';
import * as Turbo from '@hotwired/turbo';
import ujs from '@rails/ujs';
import * as bootstrap from 'bootstrap'

import AutosizeController from './controllers/autosize-controller';
import ComponentDataFetcherController from './controllers/component-data-fetcher-controller';
import FollowButtonController from './controllers/follow-button-controller';
import PostFormController from "./controllers/post-form-controller";
import RepostDropdownController from "./controllers/repost-dropdown-controller";
import RepostFormController from "./controllers/repost-form-controller";
import TimelineController from "./controllers/timeline-controller";

window.Stimulus = Application.start();
Stimulus.register('autosize', AutosizeController);
Stimulus.register('component-data-fetcher', ComponentDataFetcherController);
Stimulus.register('post-form', PostFormController);
Stimulus.register('follow-button', FollowButtonController);
Stimulus.register('repost-dropdown', RepostDropdownController);
Stimulus.register('repost-form', RepostFormController);
Stimulus.register('timeline', TimelineController);

const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
Array.from(tooltipTriggerList).map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));

ujs.start();
Turbo.start();
