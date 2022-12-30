# typed: true
# frozen_string_literal: true

class Posts::DestroyController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    profile = Profile.only_kept.find_by!(atname: params[:atname])
    post = profile.posts.find(params[:post_id])

    authorize(post, :destroy?)

    current_profile!.delete_post(post:)

    flash[:success] = t("messages.posts.deleted")
    redirect_back(fallback_location: root_path)
  end
end
