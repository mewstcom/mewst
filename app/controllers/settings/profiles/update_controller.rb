# typed: true
# frozen_string_literal: true

class Settings::Profiles::UpdateController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = ProfileForm.new(form_params)

    if @form.invalid?
      return render_as_invalid
    end

    result = UpdateProfileUseCase.new(client: v1_public_client).call(form: @form, actor: current_actor.not_nil!)

    if result.errors
      @form.add_use_case_errors(result.errors.not_nil!)
      return render_as_invalid
    end

    flash[:notice] = t("messages.profiles.updated")
    redirect_to settings_profile_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:profile_form), ActionController::Parameters).permit(:atname, :avatar_url, :description, :name)
  end

  private def render_as_invalid
    render("settings/profiles/show/call", status: :unprocessable_entity)
  end
end
