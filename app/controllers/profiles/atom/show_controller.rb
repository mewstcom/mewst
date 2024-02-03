# typed: true
# frozen_string_literal: true

class Profiles::Atom::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    profile_post = ProfilePost.fetch(
      client: v1_internal_client,
      atname: params[:atname],
      before: nil,
      after: nil
    )

    if profile_post.invalid?
      return render(status: :not_found, plain: "Not found")
    end

    @profile = profile_post.profile
    @posts = profile_post.posts

    render(formats: :atom)
  end
end
