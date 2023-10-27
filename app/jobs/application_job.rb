# typed: strict
# frozen_string_literal: true

class ApplicationJob < ActiveJob::Base
  extend T::Sig

  PRIORITY = T.let({
    high: 0,
    medium: 1,
    low: 2
  }.freeze, T::Hash[Symbol, Integer])

  queue_as :default

  queue_with_priority PRIORITY.fetch(:medium)

  retry_on StandardError, wait: :exponentially_longer, attempts: 5
end
