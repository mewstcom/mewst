# typed: strict
# frozen_string_literal: true

class FanoutPostJob < ApplicationJob
  queue_as :default

  sig { params(post_id: T::Mewst::DatabaseId).void }
  def perform(post_id:)
    input = FanoutPostService::Input.new(post_id:)
    FanoutPostService.new.call(input:)
  end
end
