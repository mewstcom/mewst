# typed: strict
# frozen_string_literal: true

class SignUpCallbacks::ShowController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    email_confirmation = EmailConfirmation.find_by(event: :sign_up, token: params[:token])

    if !email_confirmation || email_confirmation.expired?
      flash[:warning] = t("messages.authentication.sign_up_link_expired")
      return redirect_to(sign_up_path)
    end

    session[:sign_up_email] = email_confirmation.email
    session[:sign_up_back] = email_confirmation.back

    ActiveRecord::Base.transaction do
      email_confirmation.destroy
    end

    redirect_to new_account_path
  end
end
