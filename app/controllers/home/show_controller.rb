# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = PostForm.new

    post_ids = Inbox.new(current_user.profile).get_post_ids
    @posts = Post.where(id: post_ids).preload(:profile).order(created_at: :desc)
  end
end
