# typed: true
# frozen_string_literal: true

class SignIn::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = EmailConfirmationForm.new(form_params)

    if @form.invalid?
      return render("sign_in/new/call")
    end

    ActiveRecord::Base.transaction do
      ConfirmEmailService.new(form: @form).call
    end

    flash[:success] = t("messages.authentication.confirmation_email_sent")
    redirect_to sign_in_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:authentication), ActionController::Parameters)
      .permit(:email)
      .merge(event: EmailConfirmationForm::EVENT_SIGN_IN)
  end
end
