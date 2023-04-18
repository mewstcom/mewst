# typed: true
# frozen_string_literal: true

class Api::Internal::Pubsub::AddPostToHomeTimelineController < ApplicationController
  include Pubsub::Subscribable

  skip_before_action :verify_authenticity_token
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    profile = Profile.only_kept.find_by(id: message_data&.dig("profile_id"))
    post = Post.find_by(id: message_data&.dig("post_id"))

    if profile.nil? || post.nil?
      fail "Invalid message_data: #{message_data.inspect}"
    end

    profile.home_timeline.add_post(post:)

    head :no_content
  end
end
