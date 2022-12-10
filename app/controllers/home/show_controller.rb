# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @post_creator = T.must(current_profile).new_post
    @posts = T.must(current_profile).home_timeline_posts
  end
end
