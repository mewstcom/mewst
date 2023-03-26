# typed: true
# frozen_string_literal: true

class VerificationChallenges::CreateController < ApplicationController
  include Authenticatable
  include Localizable
  include VerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_verification_id

  sig { returns(T.untyped) }
  def call
    @challenge = VerificationChallenge.new(form_params)
    @challenge.verification = Verification.active.find_by(id: session[:verification_id])

    if @challenge.invalid?
      return render("verification_challenges/new/call", status: :unprocessable_entity)
    end

    @challenge.success

    redirect_to next_path(@challenge.verification)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:verification_challenge), ActionController::Parameters).permit(:challenged_code)
  end

  def next_path(verification)
    case verification.event.to_sym
    when Verification::EVENT_SIGN_UP
      new_account_path
    when Verification::EVENT_PASSWORD_RESET
      edit_password_path
    else
      root_path
    end
  end
end
