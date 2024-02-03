# typed: true
# frozen_string_literal: true

class Follows::DestroyController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = FollowForm.new(atname: params[:atname])

    if @form.invalid?
      return render(
        "follows/create/call",
        content_type: "text/vnd.turbo-stream.html",
        layout: false,
        status: :unprocessable_entity
      )
    end

    result = UnfollowProfileUseCase.new(client: v1_public_client).call(form: @form)
    @profile = result.follow.not_nil!.target_profile

    render("follows/create/call", content_type: "text/vnd.turbo-stream.html", layout: false)
  end
end
