# typed: strict
# frozen_string_literal: true

class Accounts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_confirmed_email
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @sign_up = SignUp.new(email: session[:sign_up_email])
  end

  private

  def require_confirmed_email
    unless session[:sign_up_email]
      redirect_to root_path
    end
  end
end
