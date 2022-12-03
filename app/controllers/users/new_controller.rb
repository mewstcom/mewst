# typed: true
# frozen_string_literal: true

class Users::NewController < ApplicationController
  include Authenticatable

  before_action :require_confirmed_phone_number
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number = PhoneNumber.find(session[:phone_number_id])
    @form = NewUserForm.new
  end

  sig { returns(T.untyped) }
  private def require_confirmed_phone_number
    unless session[:phone_number_id]
      redirect_to root_path
    end
  end
end
