# typed: true
# frozen_string_literal: true

class Passwords::UpdateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable
  include ControllerConcerns::EmailConfirmationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_email_confirmation

  sig { returns(T.untyped) }
  def call
    @form = PasswordResetForm.new(form_params)

    if @form.invalid?
      return render("passwords/edit/call", status: :unprocessable_entity)
    end

    UpdatePasswordUseCase.new.call(
      email: @email_confirmation.not_nil!.email.not_nil!,
      password: @form.password.not_nil!
    )

    reset_session

    flash[:notice] = t("messages.password_reset.reset_successfully_html")
    redirect_to sign_in_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:password_reset_form), ActionController::Parameters).permit(:password)
  end
end
