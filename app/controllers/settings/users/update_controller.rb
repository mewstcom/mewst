# typed: true
# frozen_string_literal: true

class Settings::Users::UpdateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = UserForm.new(form_params)

    if @form.invalid?
      return render("settings/users/show/call", status: :unprocessable_entity)
    end

    UpdateUserUseCase.new.call(
      viewer: viewer!,
      locale: @form.locale.not_nil!,
      time_zone: @form.time_zone.not_nil!
    )

    flash[:notice] = t("messages.users.updated")
    redirect_to settings_user_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:user_form), ActionController::Parameters).permit(:locale, :time_zone)
  end
end
