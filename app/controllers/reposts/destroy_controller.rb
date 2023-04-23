# typed: true
# frozen_string_literal: true

class Reposts::DestroyController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    post = Post.find(params[:post_id])
    repost = current_profile!.reposts.find_sole_by(repostable: post.postable)

    ActiveRecord::Base.transaction do
      current_profile!.delete_post(post: repost.post)
    end

    head :no_content
  end
end
