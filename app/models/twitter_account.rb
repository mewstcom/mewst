# typed: strict
# frozen_string_literal: true

class TwitterAccount < ApplicationRecord
  validate :valid_scopes

  def refresh
    twitter_client.me
    binding.irb
  end

  private

  def valid_scopes
    binding.irb
  end

  def twitter_client
    @twitter_client = Mewst::TwitterClient.new(access_token:)
  end
end
