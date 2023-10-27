# typed: strict
# frozen_string_literal: true

Rails.application.configure do
  config.good_job.max_threads = 3
  config.good_job.queues = "*"
  # https://github.com/bensheldon/good_job/tree/00f80cb127df622234c2a899e963367cc1b387df#job-priority
  config.good_job.smaller_number_is_higher_priority = true
end
