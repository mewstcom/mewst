# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @post_creator = T.must(current_profile).new_post

    post_ids = Timeline.new(T.must(current_profile)).get_post_ids
    @posts = Post.where(id: post_ids).preload(:profile).order(created_at: :desc)
  end
end
