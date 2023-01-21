# typed: true
# frozen_string_literal: true

class Profiles::ShowController < ApplicationController
  include Pagy::Backend

  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = Profile.only_kept.find_by!(atname: params[:atname])
    @pagy, @posts = pagy(@profile.posts.order(created_at: :desc))
  end
end
