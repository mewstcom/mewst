# typed: true
# frozen_string_literal: true

class SignUp::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    # TODO: Remove when the app will be in Beta
    raise if Rails.env.production?

    code = Verification.generate_code
    @verification = Verification.new(form_params.merge(code:, event: :sign_up))

    @verification.send_verification_mail

    session[:verification_id] = @verification.id
    flash[:success] = t("messages.verifications.verification_mail_sent")
    redirect_to new_verification_challenge_path
  rescue ActiveRecord::RecordInvalid
    render("sign_up/new/call", status: :unprocessable_entity)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:verification), ActionController::Parameters).permit(:email)
  end
end
