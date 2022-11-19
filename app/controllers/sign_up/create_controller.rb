# typed: strict
# frozen_string_literal: true

class SignUp::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @authentication = Authentication.new(authentication_params)

    if @authentication.invalid?
      return render "sign_up/new/call"
    end

    ActiveRecord::Base.transaction do
      @authentication.confirm_email_to_sign_up
    end

    flash[:success] = t("messages.authentication.confirmation_email_sent")
    redirect_to sign_up_path
  end

  private

  def authentication_params
    params.require(:authentication).permit(:email)
  end
end
