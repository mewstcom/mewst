# typed: true
# frozen_string_literal: true

class Api::Internal::Pubsub::FanoutPostController < ApplicationController
  include Pubsub::Subscribable

  skip_before_action :verify_authenticity_token
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    post = Post.find(message_data&.dig("post_id"))
    post.add_to_followers_home_timeline

    head :no_content
  end
end
