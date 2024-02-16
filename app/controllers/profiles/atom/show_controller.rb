# typed: true
# frozen_string_literal: true

class Profiles::Atom::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = Profile.kept.find_by!(atname: params[:atname])
    @posts = @profile.posts.kept.order(published_at: :desc).limit(15)

    render(formats: :atom)
  end
end
