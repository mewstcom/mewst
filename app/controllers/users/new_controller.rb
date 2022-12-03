# typed: true
# frozen_string_literal: true

class Users::NewController < ApplicationController
  include Authenticatable

  before_action :require_phone_number_confirmation_id
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number_confirmation = PhoneNumberConfirmation.find(session[:phone_number_confirmation_id])
    @form = NewUserForm.new
  end

  sig { returns(T.untyped) }
  private def require_phone_number_confirmation_id
    unless session[:phone_number_confirmation_id]
      redirect_to root_path
    end
  end
end
