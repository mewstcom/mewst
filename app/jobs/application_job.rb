# typed: strict
# frozen_string_literal: true

class ApplicationJob < ActiveJob::Base
  extend T::Sig

  retry_on StandardError, wait: :exponentially_longer, attempts: 5
end
