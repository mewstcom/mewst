# typed: true
# frozen_string_literal: true

class Posts::ShowController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = Profile.only_kept.find_by!(idname: params[:idname])
    @post = @profile.posts.find(params[:post_id])
  end
end
