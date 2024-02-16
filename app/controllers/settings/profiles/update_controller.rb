# typed: true
# frozen_string_literal: true

class Settings::Profiles::UpdateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = ProfileForm.new(form_params.merge(viewer: current_actor))

    if @form.invalid?
      return render("settings/profiles/show/call", status: :unprocessable_entity)
    end

    UpdateProfileUseCase.new.call(
      viewer: current_actor!,
      atname: @form.atname.not_nil!,
      avatar_url: @form.avatar_url.not_nil!,
      description: @form.description.not_nil!,
      name: @form.name.not_nil!
    )

    flash[:notice] = t("messages.profiles.updated")
    redirect_to settings_profile_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:profile_form), ActionController::Parameters).permit(:atname, :avatar_url, :description, :name)
  end
end
