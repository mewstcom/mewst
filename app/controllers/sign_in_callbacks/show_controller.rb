# typed: true
# frozen_string_literal: true

class SignInCallbacks::ShowController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    email_confirmation = EmailConfirmation.find_by(event: :sign_in, token: params[:token])

    if !email_confirmation || email_confirmation.expired?
      flash[:warning] = t("messages.authentication.sign_in_link_expired")
      return redirect_to(sign_in_path)
    end

    account = Account.find_by!(email: email_confirmation.email)

    ActiveRecord::Base.transaction do
      email_confirmation.destroy
      sign_in(account)
    end

    redirect_to home_path
  end
end
