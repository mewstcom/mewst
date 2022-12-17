# typed: true
# frozen_string_literal: true

class Users::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_phone_number_verification_challenge_id
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_verification_challenge = PhoneNumberVerificationChallenge.find(session[:phone_number_verification_challenge_id])
    idname, = form_params.values_at(:idname)
    locale = I18n.locale.to_s
    @command = Commands::CreateUser.new(phone_number_verification_challenge:, idname:, locale:)

    if @command.invalid?
      return render("users/new/call")
    end

    @command.call

    reset_session
    sign_in(@command.user!)

    redirect_to home_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:commands_create_user), ActionController::Parameters).permit(:idname)
  end

  sig { returns(T.untyped) }
  def require_phone_number_verification_challenge_id
    unless session[:phone_number_verification_challenge_id]
      redirect_to root_path
    end
  end
end
