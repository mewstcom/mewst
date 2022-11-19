# typed: strict
# frozen_string_literal: true

class Accounts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @sign_up = SignUp.new(sign_up_params)

    result = ActiveRecord::Base.transaction do
      @sign_up.create
    end

    if result.errors.any?
      return render "accounts/new/call"
    end

    reset_session
    sign_in(result.user)

    redirect_to home_path
  end

  private

  def sign_up_params
    params.require(:sign_up).permit(:email, :idname)
  end
end
