# typed: true
# frozen_string_literal: true

class Api::Internal::Pubsub::AddPostToHomeTimeline::CreateController < ApplicationController
  include Pubsub::Subscribable

  skip_before_action :verify_authenticity_token
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    profile = Profile.only_kept.find_by(id: message_data["profile_id"])
    post = Post.find_by(id: message_data["post_id"])

    if profile && post
      profile.home_timeline.add_post(post:)
    end

    head :no_content
  end
end
