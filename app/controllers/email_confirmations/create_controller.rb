# typed: true
# frozen_string_literal: true

class EmailConfirmations::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable
  include ControllerConcerns::EmailConfirmationFindable

  around_action :set_locale
  before_action :require_email_confirmation_id

  sig { returns(T.untyped) }
  def call
    @form = EmailConfirmationChallengeForm.new(form_params.merge(email_confirmation_id: session[:email_confirmation_id]))

    if @form.invalid?
      return render("email_confirmations/new/call", status: :unprocessable_entity)
    end

    result = ConfirmEmailUseCase.new.call(
      viewer:,
      email_confirmation: @form.email_confirmation!
    )

    flash_message(result.email_confirmation)
    redirect_to success_path(result.email_confirmation)
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:email_confirmation_challenge_form), ActionController::Parameters).permit(:confirmation_code)
  end

  sig { params(email_confirmation: EmailConfirmation).void }
  private def flash_message(email_confirmation)
    if email_confirmation.deserialized_event == EmailConfirmationEvent::EmailUpdate
      flash[:notice] = t("messages.email_confirmations.email_updated")
    end

    nil
  end

  sig { params(email_confirmation: EmailConfirmation).returns(String) }
  private def success_path(email_confirmation)
    event = email_confirmation.deserialized_event

    case event
    when EmailConfirmationEvent::EmailUpdate
      settings_email_path
    when EmailConfirmationEvent::PasswordReset
      edit_password_path
    when EmailConfirmationEvent::SignUp
      new_account_path
    else
      T.absurd(event)
    end
  end
end
