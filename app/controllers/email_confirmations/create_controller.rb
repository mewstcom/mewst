# typed: true
# frozen_string_literal: true

class EmailConfirmations::CreateController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable
  include EmailConfirmationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_email_confirmation_id

  sig { returns(T.untyped) }
  def call
    @form = EmailConfirmationChallengeForm.new(form_params.merge(email_confirmation_id: session[:email_confirmation_id]))

    if @form.invalid?
      return render("email_confirmations/new/call", status: :unprocessable_entity)
    end

    result = ConfirmEmailUseCase.new(client: v1_internal_client).call(form: @form)

    if result.errors
      @form.add_use_case_errors(result.errors.not_nil!)
      return render("email_confirmations/new/call", status: :unprocessable_entity)
    end

    redirect_to success_path(result.email_confirmation.not_nil!)
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:email_confirmation_challenge_form), ActionController::Parameters).permit(:confirmation_code)
  end

  sig { params(email_confirmation: EmailConfirmation).returns(String) }
  private def success_path(email_confirmation)
    event = email_confirmation.event.not_nil!

    case event
    when EmailConfirmationEvent::PasswordReset
      edit_password_path
    when EmailConfirmationEvent::SignUp
      new_account_path
    else
      T.absurd(event)
    end
  end
end
