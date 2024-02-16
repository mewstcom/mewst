# typed: true
# frozen_string_literal: true

class Checks::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = SuggestedFollowForm.new(
      source_profile: current_actor!.profile.not_nil!,
      target_atname: params[:atname]
    )

    if @form.invalid?
      return render(content_type: "text/vnd.turbo-stream.html", status: :unprocessable_entity, layout: false)
    end

    CheckSuggestedFollowUseCase.new.call(
      source_profile: @form.source_profile.not_nil!,
      target_profile: @form.target_profile.not_nil!
    )

    if current_actor!.checkable_suggested_followees.exists?
      @atname = params[:atname]
      render(content_type: "text/vnd.turbo-stream.html", layout: false)
    else
      redirect_to search_path
    end
  end
end
