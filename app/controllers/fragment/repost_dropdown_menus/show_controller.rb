# typed: true
# frozen_string_literal: true

class Fragment::RepostDropdownMenus::ShowController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @post = Post.find(params[:post_id])
    # @repost_creator = Repost::Creator.new(post_id: @post.id)
  end
end
