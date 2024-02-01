# typed: true
# frozen_string_literal: true

class Passwords::UpdateController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable
  include EmailConfirmationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_email_confirmation

  sig { returns(T.untyped) }
  def call
    @form = EditPasswordForm.new(form_params)

    if @form.invalid?
      return render("passwords/edit/call", status: :unprocessable_entity)
    end

    result = UpdatePasswordUseCase.new(client: v1_internal_client).call(
      email: @email.not_nil!,
      new_password: @form.new_password.not_nil!
    )

    if result.errors
      @form.add_use_case_errors(result.errors.not_nil!)
      return render("passwords/edit/call", status: :unprocessable_entity)
    end

    reset_session

    flash[:notice] = t("messages.password_reset.reset_successfully_html")
    redirect_to sign_in_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:edit_password_form), ActionController::Parameters).permit(:new_password)
  end
end
