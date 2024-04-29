# typed: true
# frozen_string_literal: true

class Posts::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = Profile.kept.find_by!(atname: params[:atname])
    @post = @profile.posts.kept.find(params[:post_id])
    @stamp_checker = StampChecker.new(profile: viewer&.profile, posts: [@post])
  end
end
