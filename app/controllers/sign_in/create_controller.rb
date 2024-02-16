# typed: true
# frozen_string_literal: true

class SignIn::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = SessionForm.new(form_params)

    if @form.invalid?
      return render("sign_in/new/call", status: :unprocessable_entity)
    end

    result = CreateSessionUseCase.new.call(user: @form.user.not_nil!)

    sign_in(result.actor)

    flash[:notice] = t("messages.accounts.signed_in_successfully")
    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:session_form), ActionController::Parameters).permit(:email, :password)
  end
end
