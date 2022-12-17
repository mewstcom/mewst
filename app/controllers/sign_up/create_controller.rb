# typed: true
# frozen_string_literal: true

class SignUp::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number, phone_number_origin = form_params.values_at(:phone_number, :phone_number_origin)
    confirmation_code = PhoneNumberVerificationChallenge.generate_confirmation_code
    @command = Commands::SetupPhoneNumberVerificationChallenge.new(phone_number:, phone_number_origin:, confirmation_code:)

    if @command.invalid?
      return render("sign_up/new/call")
    end

    @command.call

    session[:phone_number_verification_challenge_id] = @command.phone_number_verification_challenge!.id
    flash[:success] = t("messages.authentication.confirmation_sms_sent")
    redirect_to sign_up_phone_number_new_verification_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:commands_setup_phone_number_verification_challenge), ActionController::Parameters).permit(:phone_number, :phone_number_origin)
  end
end
