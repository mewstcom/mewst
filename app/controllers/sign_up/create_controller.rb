# typed: true
# frozen_string_literal: true

class SignUp::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    # TODO: Remove if the app will be in Beta
    raise if Rails.env.production?

    confirmation_code = PhoneNumberVerification.generate_confirmation_code
    @verification = PhoneNumberVerification.new(form_params.merge(confirmation_code:))

    @verification.save!
    SendPhoneNumberVerificationMessageJob.perform_async(@verification.id)

    session[:phone_number_verification_id] = @verification.id
    flash[:success] = t("messages.authentication.confirmation_sms_sent")
    redirect_to sign_up_verification_phone_number_new_challenge_path
  rescue ActiveRecord::RecordInvalid
    render("sign_up/new/call", status: :unprocessable_entity)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:phone_number_verification), ActionController::Parameters).permit(:phone_number, :raw_phone_number)
  end
end
