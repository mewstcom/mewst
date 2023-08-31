# typed: strict
# frozen_string_literal: true

Rails.application.configure do
  config.good_job.max_threads = 3
  config.good_job.queues = "*"
end
