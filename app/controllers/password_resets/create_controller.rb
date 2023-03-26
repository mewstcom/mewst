# typed: true
# frozen_string_literal: true

class PasswordResets::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    code = Verification.generate_code
    @verification = Verification.new(form_params.merge(code:, event: :password_reset))
    @verification.save!

    SendVerificationMailJob.perform_later(verification_id: @verification.id, locale: I18n.locale)

    session[:verification_id] = @verification.id
    flash[:success] = t("messages.verifications.verification_mail_sent")
    redirect_to new_verification_challenge_path
  rescue ActiveRecord::RecordInvalid
    render("password_resets/new/call", status: :unprocessable_entity)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:verification), ActionController::Parameters).permit(:email)
  end
end
