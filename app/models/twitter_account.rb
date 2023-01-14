# typed: strict
# frozen_string_literal: true

class TwitterAccount < ApplicationRecord
  sig { params(code: String).returns(T.self_type) }
  def self.create_from_authorization_code(code:)
    res = twitter.create_access_token(code: )
    twitter_account = profile!.build_twitter_account()
  end

  def can_tweet?

  end

  private

  def twitter
    @twitter ||= Twitter.new
  end
end
